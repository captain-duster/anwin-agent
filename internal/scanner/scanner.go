package scanner

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/captain-duster/anwin-agent/internal/client"
)

const maxFileSizeBytes = 2 * 1024 * 1024

var supportedExtensions = map[string]bool{
	".java": true, ".kt": true, ".kts": true, ".scala": true, ".sc": true, ".groovy": true, ".gvy": true, ".gradle": true,
	".py": true, ".pyw": true,
	".js": true, ".mjs": true, ".cjs": true, ".jsx": true,
	".ts": true, ".tsx": true,
	".go": true,
	".rb": true, ".rake": true, ".gemspec": true,
	".php": true,
	".cs": true, ".csx": true,
	".cpp": true, ".cc": true, ".cxx": true, ".c": true, ".h": true, ".hpp": true, ".hxx": true,
	".rs": true,
	".swift": true,
	".dart": true,
	".sql": true,
	".sh": true, ".bash": true, ".zsh": true, ".ps1": true,
	".r": true,
	".html": true, ".htm": true, ".css": true, ".scss": true, ".sass": true, ".less": true,
	".graphql": true, ".gql": true,
	".sol": true,
	".lua": true, ".pl": true, ".pm": true, ".hs": true, ".ex": true, ".exs": true, ".erl": true, ".hrl": true,
	".yaml": true, ".yml": true, ".json": true, ".toml": true, ".xml": true, ".properties": true, ".env": true,
	".tf": true, ".tfvars": true,
	".jsp": true, ".vue": true, ".svelte": true,
	".md": true, ".txt": true, ".csv": true,
}

var ignoredDirs = map[string]bool{
	".git":         true,
	".svn":         true,
	".hg":          true,
	"node_modules": true,
	"vendor":       true,
	"build":        true,
	"dist":         true,
	"out":          true,
	"__pycache__":  true,
	".idea":        true,
	".vscode":      true,
	"target":       true,
	"bin":          true,
	"obj":          true,
	".next":        true,
	".nuxt":        true,
	"coverage":     true,
	".terraform":   true,
	".gradle":      true,
	".cache":       true,
	".tox":         true,
	"venv":         true,
	".venv":        true,
	"env":          true,
	".env":         true,
	".DS_Store":    true,
	"Pods":         true,
	".dart_tool":   true,
	".pub-cache":   true,
}

var ignoredFiles = map[string]bool{
	".DS_Store":      true,
	"Thumbs.db":      true,
	".gitignore":     true,
	".gitattributes": true,
	"package-lock.json": true,
	"yarn.lock":      true,
	"pnpm-lock.yaml": true,
	"go.sum":         true,
	"Cargo.lock":     true,
	"Gemfile.lock":   true,
	"composer.lock":  true,
	"poetry.lock":    true,
}

func ScanDirectory(root string) []client.FileEntry {
	var files []client.FileEntry

	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			if isIgnoredDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		if !isSupportedFile(path, root) {
			return nil
		}

		if info.Size() > maxFileSizeBytes {
			return nil
		}

		if info.Size() == 0 {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		if isBinary(data) {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		hash := sha256Hex(data)

		files = append(files, client.FileEntry{
			RelativePath: relPath,
			Content:      string(data),
			Hash:         hash,
			Deleted:      false,
		})

		return nil
	})

	return files
}

func IsSupportedFile(path string, root string) bool {
	return isSupportedFile(path, root)
}

func IsIgnoredDir(name string) bool {
	return isIgnoredDir(name)
}

func isSupportedFile(path string, root string) bool {
	base := filepath.Base(path)

	if ignoredFiles[base] {
		return false
	}

	if strings.HasPrefix(base, ".") && base != ".env" {
		return false
	}

	ext := strings.ToLower(filepath.Ext(path))

	if base == "Dockerfile" || base == "Makefile" || base == "Rakefile" || base == "Vagrantfile" || base == "Jenkinsfile" || base == "Procfile" {
		return true
	}

	return supportedExtensions[ext]
}

func isIgnoredDir(name string) bool {
	return ignoredDirs[name]
}

func isBinary(data []byte) bool {
	check := 512
	if len(data) < check {
		check = len(data)
	}
	for i := 0; i < check; i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}

func HashFile(data []byte) string {
	return sha256Hex(data)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
