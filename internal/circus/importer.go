package circus

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/everlier/claune/internal/audio"
)

// ImportMemeSound fetches a sound from a URL and saves it to the local cache directory
// under a specified name, effectively acting as the Meme Sound Importer.
func ImportMemeSound(url, name string) error {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("invalid URL format: %q. Only http:// and https:// URLs are supported for importing meme sounds.", url)
	}
	if name == "" || name == "." || name != filepath.Base(name) {
		return fmt.Errorf("invalid import name %q", name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	defer signal.Stop(sigChan)

	go func() {
		select {
		case <-sigChan:
			cancel()
		case <-ctx.Done():
		}
	}()

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for %s: %w", url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch meme sound from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status fetching %s: %s", url, resp.Status)
	}

	cacheDir := audio.SoundCacheDir()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	tmpDest, err := os.CreateTemp(cacheDir, name+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmpDest.Name()
	
	defer func() {
		tmpDest.Close()
		os.Remove(tmpName)
	}()

	// Apply a 50MB limit to prevent downloading excessively large files
	// (Denial of Service - Disk/Memory exhaustion).
	_, err = io.Copy(tmpDest, io.LimitReader(resp.Body, 50*1024*1024))
	if err != nil {
		return fmt.Errorf("failed to save meme sound %s: %w", name, err)
	}

	if err := tmpDest.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	dest := filepath.Join(cacheDir, name)
	if err := os.Rename(tmpName, dest); err != nil {
		return fmt.Errorf("failed to rename temp file to %s: %w", dest, err)
	}

	// Clean up old cached files and partial temp files (50MB max, 100 files max)
	audio.EvictCache(cacheDir, 50*1024*1024, 100)

	return nil
}
