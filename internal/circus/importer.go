package circus

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

	resp, err := http.Get(url)
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

	dest := filepath.Join(cacheDir, name)
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	// Apply a 50MB limit to prevent downloading excessively large files
	// (Denial of Service - Disk/Memory exhaustion).
	_, err = io.Copy(out, io.LimitReader(resp.Body, 50*1024*1024))
	if err != nil {
		return fmt.Errorf("failed to save meme sound %s: %w", name, err)
	}

	return nil
}
