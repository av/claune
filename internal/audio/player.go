package audio

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/everlier/claune/internal/config"
	"github.com/gopxl/beep/mp3"
)

//go:embed sounds/*.mp3
var SoundFS embed.FS

var DefaultSoundMap = map[string]string{
	"cli:start":        "fanfare.mp3",
	"tool:start":       "clown-horn.mp3",
	"tool:success":     "applause.mp3",
	"tool:error":       "sad-trombone.mp3",
	"cli:done":         "applause.mp3",
	"tool:destructive": "maniacal-laugh.mp3",
	"tool:readonly":    "boing.mp3",
	"build:success":    "slide-whistle-up.mp3",
	"test:fail":        "sad-trombone.mp3",
	"panic":            "maniacal-laugh.mp3",
	"warn":             "boing.mp3",
}

func playMP3File(mp3Path string, volume float64, blocking bool) error {
	f, err := os.Open(mp3Path)
	if err != nil {
		return err
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		f.Close()
		return err
	}

	err = playMP3Stream(streamer, format, volume, blocking, func() {
		streamer.Close()
		f.Close()
	})
	if err != nil {
		return err
	}

	return nil
}

func SoundCacheDir() string {
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return filepath.Join(dir, "claune")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "claune")
}

func EnsureSoundCache() error {
	cacheDir := SoundCacheDir()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}
	for _, file := range DefaultSoundMap {
		dest := filepath.Join(cacheDir, file)
		data, err := SoundFS.ReadFile("sounds/" + file)
		if err != nil {
			return err
		}
		if info, err := os.Stat(dest); err == nil && info.Size() == int64(len(data)) {
			continue
		}
		if err := os.WriteFile(dest, data, 0644); err != nil {
			return err
		}
	}
	return nil
}

func cachedSoundPath(soundFile string) string {
	path := filepath.Join(SoundCacheDir(), soundFile)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

func playEmbeddedSound(soundFile string, volume float64, blocking bool) error {
	f, err := SoundFS.Open("sounds/" + soundFile)
	if err != nil {
		return err
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		f.Close()
		return err
	}

	err = playMP3Stream(streamer, format, volume, blocking, func() {
		streamer.Close()
		f.Close()
	})
	if err != nil {
		return err
	}

	return nil
}

func PlaySound(eventType string, blocking bool, c config.ClauneConfig) error {
	if c.ShouldMute() {
		return nil
	}
	volume := c.GetVolume()
	if customPath, ok := c.Sounds[eventType]; ok && customPath != "" {
		if strings.HasPrefix(customPath, "~/") {
			home, _ := os.UserHomeDir()
			customPath = filepath.Join(home, customPath[2:])
		}
		if info, err := os.Stat(customPath); err == nil && info.Size() > 0 {
			err = playMP3File(customPath, volume, blocking)
			if err != nil {
				fmt.Fprintln(os.Stderr, "🔇 Audio unavailable:", err)
			}
			return err
		}
	}
	soundFile, ok := DefaultSoundMap[eventType]
	if !ok {
		return fmt.Errorf("unknown event type: %s", eventType)
	}
	err := playEmbeddedSound(soundFile, volume, blocking)
	if err != nil {
		fmt.Fprintln(os.Stderr, "🔇 Audio unavailable:", err)
	}
	return err
}

func ShellPlayCmd(wavPath string, volume float64) string {
	return ""
}
