package audio

import (
	"embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"github.com/everlier/claune/internal/xdg"

	"github.com/everlier/claune/internal/config"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/wav"
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

func stateFilePath() string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		legacyPath := filepath.Join(home, ".claune.state.json")
		if _, err := os.Stat(legacyPath); err == nil {
			return legacyPath
		}
	}
	dir := xdg.StateHome()
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "state.json")
}

func loadState() {
	path := stateFilePath()
	data, err := os.ReadFile(path)
	if err == nil {
		// ignore errors, just use empty if unparseable
		importJSON(data)
	}
}

func saveState() {
	path := stateFilePath()
	data := exportJSON()
	os.WriteFile(path, data, 0644)
}

func importJSON(data []byte) {
	var state map[string]int
	if err := json.Unmarshal(data, &state); err == nil && state != nil {
		for k, v := range state {
			rrIndex[k] = v
		}
	}
}

func exportJSON() []byte {
	data, _ := json.Marshal(rrIndex)
	return data
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func playMP3File(mp3Path string, volume float64, blocking bool) error {
	f, err := os.Open(mp3Path)
	if err != nil {
		return err
	}

	var streamer beep.StreamSeekCloser
	var format beep.Format

	if strings.HasSuffix(strings.ToLower(mp3Path), ".wav") {
		streamer, format, err = wav.Decode(f)
	} else {
		streamer, format, err = mp3.Decode(f)
	}

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
	dir := xdg.CacheHome()
	os.MkdirAll(dir, 0755)
	return dir
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

func pickSound(stateKey string, sounds []string, strategy string) string {
	if len(sounds) == 0 {
		return ""
	}
	if len(sounds) == 1 {
		return sounds[0]
	}

	if strategy == "round_robin" {
		rrMutex.Lock()
		defer rrMutex.Unlock()

		// Inter-process lock using O_EXCL
		path := stateFilePath()
		lockPath := path + ".lock"
		locked := false
		for i := 0; i < 50; i++ {
			f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
			if err == nil {
				f.Close()
				locked = true
				break
			}
			if info, err := os.Stat(lockPath); err == nil {
				if time.Since(info.ModTime()) > 2*time.Second {
					os.Remove(lockPath)
					continue
				}
			}
			time.Sleep(10 * time.Millisecond)
		}
		if locked {
			defer os.Remove(lockPath)
		} else {
			// Fallback if we cannot acquire inter-process lock
			return sounds[rand.Intn(len(sounds))]
		}

		loadState()
		idx := rrIndex[stateKey]
		selected := sounds[idx%len(sounds)]
		rrIndex[stateKey] = (idx + 1) % len(sounds)
		saveState()
		return selected
	}

	// Default to random
	return sounds[rand.Intn(len(sounds))]
}

func PlaySoundWithStrategy(eventType string, overrideStrategy string, blocking bool, c config.ClauneConfig) error {
	if c.ShouldMute() {
		return nil
	}
	volume := c.GetVolume()

	// Check custom configured sounds first
	if customConfig, ok := c.Sounds[eventType]; ok && len(customConfig.Paths) > 0 {
		strategy := overrideStrategy
		if strategy == "" {
			strategy = customConfig.Strategy
			if strategy == "" {
				strategy = c.Strategy // global fallback
				if strategy == "" {
					strategy = "random" // default
				}
			}
		}

		customPath := pickSound(eventType+":custom", customConfig.Paths, strategy)
		if customPath != "" {
			if strings.HasPrefix(customPath, "~/") {
				home, err := os.UserHomeDir()
				if err != nil || home == "" {
					home = os.TempDir()
				}
				customPath = filepath.Join(home, customPath[2:])
			}
			if info, err := os.Stat(customPath); err == nil && info.Size() > 0 {
				err = playMP3File(customPath, volume, blocking)
				if err == nil {
					return nil
				}
				fmt.Fprintf(os.Stderr, "Warning: failed to play custom sound %q for event %q: %v\n", customPath, eventType, err)
			} else {
				if err == nil {
					err = fmt.Errorf("file is empty")
				}
				fmt.Fprintf(os.Stderr, "Warning: invalid custom sound path %q for event %q: %v\n", customPath, eventType, err)
			}
		}
	}

	// Fallback to default sounds
	soundFiles, ok := DefaultSoundMap[eventType]
	if !ok || len(soundFiles) == 0 {
		return fmt.Errorf("unknown event type: %s\nValid types: %s", eventType, validEventTypes())
	}

	strategy := overrideStrategy
	if strategy == "" {
		if customConfig, ok := c.Sounds[eventType]; ok && customConfig.Strategy != "" {
			strategy = customConfig.Strategy
		} else if c.Strategy != "" {
			strategy = c.Strategy
		} else {
			strategy = "random"
		}
	}

	soundFile := pickSound(eventType, soundFiles, strategy)
	err := playEmbeddedSound(soundFile, volume, blocking)
	return err
}

func PlaySound(eventType string, blocking bool, c config.ClauneConfig) error {
	return PlaySoundWithStrategy(eventType, "", blocking, c)
}

func validEventTypes() string {
	keys := make([]string, 0, len(DefaultSoundMap))
	for k := range DefaultSoundMap {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}
