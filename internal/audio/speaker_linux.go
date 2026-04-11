//go:build linux

package audio

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/wav"
)

func initSpeaker(sampleRate beep.SampleRate) error {
	return nil
}

func playMP3Stream(streamer beep.StreamSeekCloser, format beep.Format, volume float64, blocking bool, cleanup func()) error {
	cacheDir := SoundCacheDir()
	os.MkdirAll(cacheDir, 0755)

	var ctrl beep.Streamer = streamer
	if volume != 1.0 {
		volLog := 0.0
		if volume > 0.001 {
			volLog = math.Log2(volume)
		}
		ctrl = &effects.Volume{
			Streamer: streamer,
			Base:     2,
			Volume:   volLog,
			Silent:   volume <= 0.001,
		}
	}

	doPlay := func() error {
		tmpFile, err := os.CreateTemp(cacheDir, "play-*.tmp.wav")
		if err != nil {
			if cleanup != nil {
				cleanup()
			}
			return fmt.Errorf("failed to create temp file: %w", err)
		}

		err = wav.Encode(tmpFile, ctrl, format)
		tmpFile.Close()

		if cleanup != nil {
			cleanup()
		}

		if err != nil {
			os.Remove(tmpFile.Name())
			return fmt.Errorf("failed to encode stream to wav: %w", err)
		}

		var bin string
		var args []string
		if path, err := exec.LookPath("paplay"); err == nil {
			bin = path
			args = []string{tmpFile.Name()}
		} else if path, err := exec.LookPath("pw-play"); err == nil {
			bin = path
			args = []string{tmpFile.Name()}
		} else if path, err := exec.LookPath("aplay"); err == nil {
			bin = path
			args = []string{"-q", tmpFile.Name()}
		} else {
			os.Remove(tmpFile.Name())
			return fmt.Errorf("no audio backend found (checked paplay, pw-play, aplay)")
		}

		cmd := exec.Command(bin, args...)
		err = cmd.Run()
		os.Remove(tmpFile.Name())
		if err != nil {
			return fmt.Errorf("%s failed: %w", filepath.Base(bin), err)
		}
		return nil
	}

	if blocking {
		return doPlay()
	}

	go doPlay()
	return nil
}
