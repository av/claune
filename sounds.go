package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed sounds/*.wav
var soundFS embed.FS

var defaultSoundMap = map[string]string{
	"cli:start":    "fanfare.wav",
	"tool:start":   "drumroll.wav",
	"tool:success": "tada.wav",
	"tool:error":   "sad-trombone.wav",
	"cli:done":     "applause.wav",
}

// ClauneConfig represents ~/.claune.json
type ClauneConfig struct {
	Mute   *bool                  `json:"mute,omitempty"`
	Volume *float64               `json:"volume,omitempty"`
	Sounds map[string]string      `json:"sounds,omitempty"`
	Extra  map[string]interface{} `json:"-"` // non-sound keys for merging
}

func getConfig() ClauneConfig {
	config := ClauneConfig{
		Sounds: make(map[string]string),
	}

	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".claune.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return config
	}

	// First parse known fields
	json.Unmarshal(data, &config)

	// Also parse all fields for merging
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)
	config.Extra = make(map[string]interface{})
	for k, v := range raw {
		if k != "mute" && k != "volume" && k != "sounds" {
			config.Extra[k] = v
		}
	}

	if config.Sounds == nil {
		config.Sounds = make(map[string]string)
	}

	return config
}

func shouldMute(config ClauneConfig) bool {
	if config.Mute != nil {
		return *config.Mute
	}
	// Smart mute: auto-mute between 23:00 and 07:00 if mute not explicitly set
	now := time.Now()
	hour := now.Hour()
	return hour >= 23 || hour < 7
}

func getVolume(config ClauneConfig) float64 {
	if config.Volume != nil {
		return *config.Volume
	}
	return 1.0
}

// findAudioPlayer returns the command to use for playing WAV files
func findAudioPlayer() (string, []string) {
	// Try paplay first (PulseAudio/PipeWire)
	if path, err := exec.LookPath("paplay"); err == nil {
		return path, nil
	}
	// Try pw-play (PipeWire native)
	if path, err := exec.LookPath("pw-play"); err == nil {
		return path, nil
	}
	// Try aplay (ALSA)
	if path, err := exec.LookPath("aplay"); err == nil {
		return path, []string{"-q"}
	}
	// macOS
	if path, err := exec.LookPath("afplay"); err == nil {
		return path, nil
	}
	return "", nil
}

// playWAVFile plays a WAV file using system audio tools
// If blocking is true, waits for playback to finish
func playWAVFile(wavPath string, volume float64, blocking bool) error {
	player, extraArgs := findAudioPlayer()
	if player == "" {
		return fmt.Errorf("no audio player found")
	}

	args := make([]string, 0, len(extraArgs)+1)
	args = append(args, extraArgs...)

	// paplay and afplay support volume flags
	baseName := filepath.Base(player)
	if volume != 1.0 {
		switch baseName {
		case "paplay":
			// paplay --volume takes a value where 65536 = 100%
			vol := int(volume * 65536)
			args = append(args, fmt.Sprintf("--volume=%d", vol))
		case "afplay":
			// afplay -v takes a float
			args = append(args, "-v", fmt.Sprintf("%.2f", volume))
		}
	}

	args = append(args, wavPath)

	cmd := exec.Command(player, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if blocking {
		return cmd.Run()
	}
	return cmd.Start()
}

// playEmbeddedSound writes embedded WAV to temp file and plays it
func playEmbeddedSound(soundFile string, volume float64, blocking bool) error {
	data, err := soundFS.ReadFile("sounds/" + soundFile)
	if err != nil {
		return err
	}

	// Write to temp file
	tmpFile, err := os.CreateTemp("", "claune-*.wav")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}
	tmpFile.Close()

	if blocking {
		defer os.Remove(tmpPath)
		return playWAVFile(tmpPath, volume, true)
	}

	// Non-blocking: clean up after playback
	go func() {
		playWAVFile(tmpPath, volume, true)
		os.Remove(tmpPath)
	}()
	return nil
}

// playSound plays the sound for the given event type
func playSound(eventType string, blocking bool) {
	config := getConfig()
	if shouldMute(config) {
		return
	}

	volume := getVolume(config)

	// Check for custom sound override
	if customPath, ok := config.Sounds[eventType]; ok && customPath != "" {
		// Expand ~ in path
		if strings.HasPrefix(customPath, "~/") {
			home, _ := os.UserHomeDir()
			customPath = filepath.Join(home, customPath[2:])
		}
		// Check file exists and is non-empty
		info, err := os.Stat(customPath)
		if err != nil || info.Size() == 0 {
			return
		}
		playWAVFile(customPath, volume, blocking)
		return
	}

	// Use default embedded sound
	soundFile, ok := defaultSoundMap[eventType]
	if !ok {
		return
	}

	playEmbeddedSound(soundFile, volume, blocking)
}

func handlePlaySubcommand(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: claune play <sound_type>")
		fmt.Fprintln(os.Stderr, "Types: cli:start, tool:start, tool:success, tool:error, cli:done")
		os.Exit(1)
	}

	playSound(args[0], true) // blocking in play mode
}
