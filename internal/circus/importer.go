package circus

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/everlier/claune/internal/audio"
)

// ImportMemeSound fetches a sound from a URL and saves it to the local cache directory
// under a specified name, effectively acting as the Meme Sound Importer.
func ImportMemeSound(url, name string) error {
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

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save meme sound %s: %w", name, err)
	}

	fmt.Printf("🤡 Successfully imported meme sound to %s\n", dest)
	return nil
}
