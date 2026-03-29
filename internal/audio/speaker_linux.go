//go:build linux

package audio

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/wav"
)

func initSpeaker(sampleRate beep.SampleRate) error {
	return nil
}

func playMP3Stream(streamer beep.StreamSeekCloser, format beep.Format, volume float64, blocking bool, cleanup func()) error {
	cacheDir := SoundCacheDir()
	os.MkdirAll(cacheDir, 0755)

	doPlay := func() error {
		tmpFile, err := os.CreateTemp(cacheDir, "play-*.wav")
		if err != nil {
			if cleanup != nil {
				cleanup()
			}
			return fmt.Errorf("failed to create temp file: %w", err)
		}

		err = wav.Encode(tmpFile, streamer, format)
		tmpFile.Close()

		if cleanup != nil {
			cleanup()
		}

		if err != nil {
			os.Remove(tmpFile.Name())
			return fmt.Errorf("failed to encode stream to wav: %w", err)
		}

		cmd := exec.Command("aplay", "-q", tmpFile.Name())
		err = cmd.Run()
		os.Remove(tmpFile.Name())
		if err != nil {
			return fmt.Errorf("aplay failed: %w", err)
		}
		return nil
	}

	if blocking {
		return doPlay()
	}

	go doPlay()
	return nil
}
