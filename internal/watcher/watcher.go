package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/anwin/agent/internal/client"
	"github.com/anwin/agent/internal/logger"
	"github.com/anwin/agent/internal/scanner"
	"github.com/fsnotify/fsnotify"
)

const (
	debounceDelay = 2 * time.Second
	heartbeatEvery = 60 * time.Second
)

type Watcher struct {
	watchPath   string
	projectID   string
	client      *client.Client
	pending     map[string]client.FileEntry
	fileHashes  map[string]string
	mu          sync.Mutex
	timer       *time.Timer
	done        chan struct{}
}

func New(watchPath, projectID string, c *client.Client) *Watcher {
	return &Watcher{
		watchPath:  watchPath,
		projectID:  projectID,
		client:     c,
		pending:    make(map[string]client.FileEntry),
		fileHashes: make(map[string]string),
		done:       make(chan struct{}),
	}
}

func (w *Watcher) Stop() {
	close(w.done)
}

func (w *Watcher) Start() error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to initialize file watcher: %w", err)
	}
	defer fsw.Close()

	if err := w.registerDirs(fsw, w.watchPath); err != nil {
		return fmt.Errorf("failed to register directories: %w", err)
	}

	heartbeat := time.NewTicker(heartbeatEvery)
	defer heartbeat.Stop()

	for {
		select {
		case <-w.done:
			logger.Info("Watcher stopping, flushing pending changes")
			w.flushNow()
			return nil

		case event, ok := <-fsw.Events:
			if !ok {
				return nil
			}
			w.handleEvent(fsw, event)

		case err, ok := <-fsw.Errors:
			if !ok {
				return nil
			}
			logger.Warn("Watcher error", "error", err.Error())

		case <-heartbeat.C:
			if !w.client.Ping() {
				logger.Warn("Heartbeat failed — server unreachable, will retry on next change")
			}
		}
	}
}

func (w *Watcher) handleEvent(fsw *fsnotify.Watcher, event fsnotify.Event) {
	info, statErr := os.Stat(event.Name)

	if statErr == nil && info.IsDir() {
		if event.Op&fsnotify.Create != 0 && !scanner.IsIgnoredDir(event.Name) {
			_ = w.registerDirs(fsw, event.Name)
		}
		return
	}

	if !scanner.IsSupportedFile(event.Name) {
		return
	}

	relPath, err := filepath.Rel(w.watchPath, event.Name)
	if err != nil {
		return
	}
	relPath = filepath.ToSlash(relPath)

	w.mu.Lock()
	defer w.mu.Unlock()

	if event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
		logger.Info("File removed", "file", relPath)
		delete(w.fileHashes, event.Name)
		w.pending[event.Name] = client.FileEntry{
			RelativePath: relPath,
			Deleted:      true,
		}
		w.scheduleSend()
		return
	}

	if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
		content, err := os.ReadFile(event.Name)
		if err != nil {
			return
		}

		if info != nil && info.Size() > 1024*1024 {
			return
		}

		hash := scanner.HashBytes(content)
		if w.fileHashes[event.Name] == hash {
			return
		}

		eventName := "modified"
		if event.Op&fsnotify.Create != 0 {
			eventName = "created"
		}

		logger.Info("Change detected", "file", relPath, "event", eventName)

		w.fileHashes[event.Name] = hash
		w.pending[event.Name] = client.FileEntry{
			RelativePath: relPath,
			Content:      string(content),
			Hash:         hash,
			Deleted:      false,
		}
		w.scheduleSend()
	}
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

	start := time.Now()
	logger.Info("Syncing changes", "files", len(files))

	if err := w.client.SyncFiles(w.projectID, files, false); err != nil {
		logger.Error("Sync failed", "error", err.Error())
	} else {
		logger.Info("Sync complete", "files", len(files), "duration", fmt.Sprintf("%.2fs", time.Since(start).Seconds()))
	}
}

func (w *Watcher) flushNow() {
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

	_ = w.client.SyncFiles(w.projectID, files, false)
}

func (w *Watcher) registerDirs(fsw *fsnotify.Watcher, root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if scanner.IsIgnoredDir(path) {
				return filepath.SkipDir
			}
			return fsw.Add(path)
		}
		return nil
	})
}
