package audio

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/everlier/claune/internal/config"
)

//go:embed sounds/*.wav
var SoundFS embed.FS

var DefaultSoundMap = map[string]string{
	"cli:start":        "fanfare.wav",
	"tool:start":       "clown-horn.wav", // Circus meme
	"tool:success":     "tada.wav",
	"tool:error":       "sad-trombone.wav", // Circus meme
	"cli:done":         "applause.wav",
	"tool:destructive": "maniacal-laugh.wav", // Circus meme
	"tool:readonly":    "boing.wav",          // Circus meme
	"build:success":    "slide-whistle-up.wav", // Circus meme
	"test:fail":        "sad-trombone.wav",
	"panic":            "maniacal-laugh.wav",
	"warn":             "boing.wav",
}

func findAudioPlayer() (string, []string) {
	if path, err := exec.LookPath("paplay"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("pw-play"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("aplay"); err == nil {
		return path, []string{"-q"}
	}
	if path, err := exec.LookPath("afplay"); err == nil {
		return path, nil
	}
	return "", nil
}

func playWAVFile(wavPath string, volume float64, blocking bool) error {
	player, extraArgs := findAudioPlayer()
	if player == "" {
		return fmt.Errorf("no audio player found")
	}

	args := make([]string, 0, len(extraArgs)+1)
	args = append(args, extraArgs...)

	baseName := filepath.Base(player)
	if volume != 1.0 {
		switch baseName {
		case "paplay":
			vol := int(volume * 65536)
			args = append(args, fmt.Sprintf("--volume=%d", vol))
		case "afplay":
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

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd.Start()
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
	if cached := cachedSoundPath(soundFile); cached != "" {
		return playWAVFile(cached, volume, blocking)
	}
	data, err := SoundFS.ReadFile("sounds/" + soundFile)
	if err != nil {
		return err
	}
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
	}
	return playWAVFile(tmpPath, volume, blocking)
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
			err = playWAVFile(customPath, volume, blocking)
			if err != nil && err.Error() == "no audio player found" {
				fmt.Fprintln(os.Stderr, "🔇 Audio unavailable: no supported audio player found (paplay, pw-play, aplay, afplay)")
			}
			return err
		}
	}
	soundFile, ok := DefaultSoundMap[eventType]
	if !ok {
		return fmt.Errorf("unknown event type: %s", eventType)
	}
	err := playEmbeddedSound(soundFile, volume, blocking)
	if err != nil && err.Error() == "no audio player found" {
		fmt.Fprintln(os.Stderr, "🔇 Audio unavailable: no supported audio player found (paplay, pw-play, aplay, afplay)")
	}
	return err
}

func ShellPlayCmd(wavPath string, volume float64) string {
	player, extraArgs := findAudioPlayer()
	if player == "" {
		return ""
	}
	cmd := player
	for _, a := range extraArgs {
		cmd += " " + a
	}
	if volume != 1.0 {
		base := filepath.Base(player)
		switch base {
		case "paplay":
			cmd += fmt.Sprintf(" --volume=%d", int(volume*65536))
		case "afplay":
			cmd += fmt.Sprintf(" -v %.2f", volume)
		}
	}
	cmd += " " + wavPath
	return cmd
}
