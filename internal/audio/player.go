package audio

import (
	"embed"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/everlier/claune/internal/config"
	"github.com/gopxl/beep/mp3"
)

//go:embed sounds/*.mp3
var SoundFS embed.FS

var DefaultSoundMap = map[string][]string{
	"cli:start":        {"cli-start.mp3"},
	"tool:start":       {"tool-start.mp3"},
	"tool:success":     {"minecraft-click.mp3", "anime-wow.mp3", "yippee.mp3"},
	"tool:error":       {"error.mp3"},
	"cli:done":         {"success.mp3"},
	"tool:destructive": {"boom.mp3"},
	"tool:readonly":    {"bonk.mp3"},
	"build:success":    {"coin.mp3"},
	"build:fail":       {"oof.mp3"},
	"test:fail":        {"fart.mp3"},
	"panic":            {"boom.mp3"},
	"warn":             {"oof.mp3"},
}

var (
	rrMutex sync.Mutex
	rrIndex = make(map[string]int)
)

func init() {
	rand.Seed(time.Now().UnixNano())
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
	for _, files := range DefaultSoundMap {
		for _, file := range files {
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

func pickSound(eventType string, sounds []string, strategy string) string {
	if len(sounds) == 0 {
		return ""
	}
	if len(sounds) == 1 {
		return sounds[0]
	}

	if strategy == "round_robin" {
		rrMutex.Lock()
		defer rrMutex.Unlock()
		idx := rrIndex[eventType]
		selected := sounds[idx%len(sounds)]
		rrIndex[eventType] = (idx + 1) % len(sounds)
		return selected
	}

	// Default to random
	return sounds[rand.Intn(len(sounds))]
}

func PlaySound(eventType string, blocking bool, c config.ClauneConfig) error {
	if c.ShouldMute() {
		return nil
	}
	volume := c.GetVolume()
	
	// Check custom configured sounds first
	if customConfig, ok := c.Sounds[eventType]; ok && len(customConfig.Paths) > 0 {
		strategy := customConfig.Strategy
		if strategy == "" {
			strategy = "random" // default
		}
		customPath := pickSound(eventType, customConfig.Paths, strategy)
		if customPath != "" {
			if strings.HasPrefix(customPath, "~/") {
				home, _ := os.UserHomeDir()
				customPath = filepath.Join(home, customPath[2:])
			}
			if info, err := os.Stat(customPath); err == nil && info.Size() > 0 {
				err = playMP3File(customPath, volume, blocking)
				return err
			}
		}
	}
	
	// Fallback to default sounds
	soundFiles, ok := DefaultSoundMap[eventType]
	if !ok || len(soundFiles) == 0 {
		return fmt.Errorf("unknown event type: %s\nValid types: %s", eventType, validEventTypes())
	}
	
	soundFile := pickSound(eventType, soundFiles, "random")
	err := playEmbeddedSound(soundFile, volume, blocking)
	return err
}

func validEventTypes() string {
	keys := make([]string, 0, len(DefaultSoundMap))
	for k := range DefaultSoundMap {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}
