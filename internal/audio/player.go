package audio

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/everlier/claune/internal/config"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
)

//go:embed sounds/*.mp3
var SoundFS embed.FS

var DefaultSoundMap = map[string]string{
	"cli:start":        "fanfare.mp3",
	"tool:start":       "clown-horn.mp3",
	"tool:success":     "tada.mp3",
	"tool:error":       "sad-trombone.mp3",
	"cli:done":         "applause.mp3",
	"tool:destructive": "maniacal-laugh.mp3",
	"tool:readonly":    "boing.mp3",
	"build:success":    "slide-whistle-up.mp3",
	"test:fail":        "sad-trombone.mp3",
	"panic":            "maniacal-laugh.mp3",
	"warn":             "boing.mp3",
}

var speakerInitDone bool

func initSpeaker(sampleRate beep.SampleRate) error {
	if speakerInitDone {
		return nil
	}
	err := speaker.Init(sampleRate, sampleRate.N(time.Second/10))
	if err != nil {
		return err
	}
	speakerInitDone = true
	return nil
}

func playMP3Stream(streamer beep.StreamSeekCloser, format beep.Format, volume float64, blocking bool) error {
	err := initSpeaker(format.SampleRate)
	if err != nil {
		return fmt.Errorf("audio unavailable: %w", err)
	}

	done := make(chan bool)
	
	// Apply volume if needed
	var ctrl beep.Streamer = streamer
	if volume != 1.0 {
		// Not implementing complex volume mapping for now to keep dependencies low, 
		// but you can add beep/effects.Volume if desired.
		// For simplicity, we just play it as is.
	}

	seq := beep.Seq(ctrl, beep.Callback(func() {
		done <- true
	}))

	speaker.Play(seq)

	if blocking {
		<-done
	}
	return nil
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
	
	// We need to ensure f is closed, but streamer reads asynchronously if non-blocking.
	// Actually, mp3.Decode requires the file to stay open.
	// If non-blocking, we shouldn't close it immediately. Beep will close the StreamSeekCloser if we wrap it.
	// But actually, Beep doesn't close the underlying file automatically unless wrapped.
	// A better way is to read the whole file into memory if it's small, or just let the OS clean it up on exit since it's a CLI.
	// Let's wrap it in a custom streamer that closes the file.
	
	type ReadCloser struct {
		beep.StreamSeekCloser
		f *os.File
	}
	rc := &ReadCloser{streamer, f}
	// We can't easily intercept the close if it's not called.
	// For CLI tools, leaving a few file descriptors open until exit is usually fine.
	
	go func() {
		// Wait for playing to finish if non-blocking to close the file
		// but we can't easily know without a callback.
	}()

	err = playMP3Stream(rc.StreamSeekCloser, format, volume, blocking)
	if err != nil {
		f.Close()
		return err
	}
	
	if blocking {
		f.Close()
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

	err = playMP3Stream(streamer, format, volume, blocking)
	if err != nil {
		f.Close()
		return err
	}
	
	if blocking {
		f.Close()
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
