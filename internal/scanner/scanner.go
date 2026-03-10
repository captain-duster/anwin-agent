package scanner

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/anwin/agent/internal/client"
)

const (
	maxFileSizeBytes = 1024 * 1024
	batchSize        = 50
)

var ignoredDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	"target":       true,
	"build":        true,
	"dist":         true,
	".idea":        true,
	".vscode":      true,
	"__pycache__":  true,
	".gradle":      true,
	".mvn":         true,
	"vendor":       true,
	"coverage":     true,
	".nyc_output":  true,
	".next":        true,
	".nuxt":        true,
	"out":          true,
	".cache":       true,
	"tmp":          true,
	"temp":         true,
	"logs":         true,
}

var supportedExtensions = map[string]bool{
	".java":        true,
	".ts":          true,
	".tsx":         true,
	".js":          true,
	".jsx":         true,
	".py":          true,
	".go":          true,
	".rs":          true,
	".kt":          true,
	".kts":         true,
	".xml":         true,
	".yml":         true,
	".yaml":        true,
	".json":        true,
	".sql":         true,
	".properties":  true,
	".gradle":      true,
	".tf":          true,
	".cs":          true,
	".cpp":         true,
	".c":           true,
	".h":           true,
	".rb":          true,
	".php":         true,
	".swift":       true,
	".scala":       true,
	".md":          true,
	".toml":        true,
	".env.example": true,
	".sh":          true,
	".dockerfile":  true,
}

type ScanResult struct {
	Batches  [][]client.FileEntry
	Total    int
	Skipped  int
}

func ScanDirectory(rootPath string) (*ScanResult, error) {
	result := &ScanResult{}
	var currentBatch []client.FileEntry

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			if IsIgnoredDir(path) {
				return filepath.SkipDir
			}
			return nil
		}

		if !IsSupportedFile(path) {
			result.Skipped++
			return nil
		}

		if info.Size() > maxFileSizeBytes {
			result.Skipped++
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			result.Skipped++
			return nil
		}

		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return nil
		}

		entry := client.FileEntry{
			RelativePath: filepath.ToSlash(relPath),
			Content:      string(content),
			Hash:         HashBytes(content),
			Deleted:      false,
		}

		currentBatch = append(currentBatch, entry)
		result.Total++

		if len(currentBatch) >= batchSize {
			result.Batches = append(result.Batches, currentBatch)
			currentBatch = nil
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(currentBatch) > 0 {
		result.Batches = append(result.Batches, currentBatch)
	}

	return result, nil
}

func IsIgnoredDir(path string) bool {
	base := filepath.Base(path)
	return ignoredDirs[base] || strings.HasPrefix(base, ".")
}

func IsSupportedFile(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	ext := strings.ToLower(filepath.Ext(path))

	if name == "dockerfile" || name == "makefile" || name == "jenkinsfile" {
		return true
	}
	if !supportedExtensions[ext] {
		return false
	}

	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, part := range parts {
		if ignoredDirs[part] {
			return false
		}
	}
	return true
}

func HashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
