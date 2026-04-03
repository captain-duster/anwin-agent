package watcher

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/captain-duster/anwin-agent/internal/client"
	"github.com/captain-duster/anwin-agent/internal/scanner"
	"github.com/fsnotify/fsnotify"
)

const debounceDelay = 2 * time.Second
const maxFileSizeBytes = 2 * 1024 * 1024
const maxBatchSize = 200

type Watcher struct {
	root        string
	anwinClient *client.AnwinClient
	pending     map[string]client.FileEntry
	mu          sync.Mutex
	timer       *time.Timer
	hashes      map[string]string
}

func New(root string, anwinClient *client.AnwinClient) *Watcher {
	return &Watcher{
		root:        root,
		anwinClient: anwinClient,
		pending:     make(map[string]client.FileEntry),
		hashes:      make(map[string]string),
	}
}

func (w *Watcher) Start() {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("  [%s] ✗ Cannot start file watcher: %s\n", ts(), err.Error())
		return
	}
	defer fsw.Close()

	if err := w.registerDirs(fsw); err != nil {
		fmt.Printf("  [%s] ✗ Cannot register directories: %s\n", ts(), err.Error())
		return
	}

	fmt.Printf("  [%s] Watching for changes...\n", ts())

	for {
		select {
		case event, ok := <-fsw.Events:
			if !ok {
				return
			}
			w.handleEvent(event, fsw)

		case err, ok := <-fsw.Errors:
			if !ok {
				return
			}
			fmt.Printf("  [%s] Watcher error: %s\n", ts(), err.Error())
		}
	}
}

func (w *Watcher) handleEvent(event fsnotify.Event, fsw *fsnotify.Watcher) {
	path := event.Name

	info, statErr := os.Stat(path)

	if statErr == nil && info.IsDir() {
		if event.Has(fsnotify.Create) && !scanner.IsIgnoredDir(info.Name()) {
			_ = fsw.Add(path)
			w.scanNewDir(path)
		}
		return
	}

	if !scanner.IsSupportedFile(path, w.root) {
		return
	}

	relPath, err := filepath.Rel(w.root, path)
	if err != nil {
		return
	}
	relPath = filepath.ToSlash(relPath)

	w.mu.Lock()
	defer w.mu.Unlock()

	if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
		w.pending[relPath] = client.FileEntry{
			RelativePath: relPath,
			Content:      "",
			Hash:         "",
			Deleted:      true,
		}
		delete(w.hashes, relPath)
		fmt.Printf("  [%s] ✗ %s\n", ts(), relPath)
		w.scheduleSend()
		return
	}

	if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
		if statErr != nil {
			return
		}

		if info.Size() > maxFileSizeBytes || info.Size() == 0 {
			return
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return
		}

		hash := hashBytes(data)

		if oldHash, exists := w.hashes[relPath]; exists && oldHash == hash {
			return
		}

		w.hashes[relPath] = hash

		w.pending[relPath] = client.FileEntry{
			RelativePath: relPath,
			Content:      string(data),
			Hash:         hash,
			Deleted:      false,
		}

		fmt.Printf("  [%s] ✦ %s\n", ts(), relPath)
		w.scheduleSend()
	}
}

func (w *Watcher) scanNewDir(dirPath string) {
	_ = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if scanner.IsIgnoredDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !scanner.IsSupportedFile(path, w.root) {
			return nil
		}
		if info.Size() > maxFileSizeBytes || info.Size() == 0 {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(w.root, path)
		if err != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)
		hash := hashBytes(data)

		w.mu.Lock()
		w.hashes[relPath] = hash
		w.pending[relPath] = client.FileEntry{
			RelativePath: relPath,
			Content:      string(data),
			Hash:         hash,
			Deleted:      false,
		}
		w.mu.Unlock()

		return nil
	})

	w.mu.Lock()
	w.scheduleSend()
	w.mu.Unlock()
}

func (w *Watcher) scheduleSend() {
	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(debounceDelay, w.flush)
}

func (w *Watcher) flush() {
	w.mu.Lock()
	if len(w.pending) == 0 {
		w.mu.Unlock()
		return
	}

	files := make([]client.FileEntry, 0, len(w.pending))
	for _, f := range w.pending {
		files = append(files, f)
	}
	w.pending = make(map[string]client.FileEntry)
	w.mu.Unlock()

	batches := client.SplitBatches(files, maxBatchSize)
	totalFiles := len(files)

	fmt.Printf("  [%s] Syncing %d file(s)...\n", ts(), totalFiles)

	for i, batch := range batches {
		if err := w.anwinClient.SyncWithRetry(batch, false, 3); err != nil {
			fmt.Printf("  [%s] ✗ Sync failed (batch %d/%d): %s\n", ts(), i+1, len(batches), err.Error())
		}
	}

	fmt.Printf("  [%s] Sync complete.\n", ts())
}

func (w *Watcher) registerDirs(fsw *fsnotify.Watcher) error {
	return filepath.Walk(w.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if scanner.IsIgnoredDir(info.Name()) {
				return filepath.SkipDir
			}
			return fsw.Add(path)
		}
		return nil
	})
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func ts() string {
	return time.Now().Format("15:04:05")
}
