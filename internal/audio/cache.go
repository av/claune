package audio

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// EvictCache removes leftover temporary files and enforces a maximum cache size.
// It uses a simple oldest-first cleanup strategy.
func EvictCache(cacheDir string, maxBytes int64, maxFiles int) error {
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	type fileInfo struct {
		path string
		info os.FileInfo
	}

	var files []fileInfo
	var totalSize int64

	now := time.Now()

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		path := filepath.Join(cacheDir, entry.Name())

		// Clean up stale temporary files (older than 1 hour)
		if strings.Contains(entry.Name(), ".tmp") {
			if now.Sub(info.ModTime()) > time.Hour {
				os.Remove(path)
			}
			continue
		}

		if !info.Mode().IsRegular() {
			continue
		}

		files = append(files, fileInfo{path: path, info: info})
		totalSize += info.Size()
	}

	// Sort files by modification time (oldest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].info.ModTime().Before(files[j].info.ModTime())
	})

	for _, f := range files {
		// Stop if we are back within limits
		if totalSize <= maxBytes && len(files) <= maxFiles {
			break
		}

		if err := os.Remove(f.path); err == nil {
			totalSize -= f.info.Size()
		}
		// decrease length count manually for condition check above
		files = files[1:]
	}

	return nil
}