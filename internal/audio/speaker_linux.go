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
	// We'll write the decoded audio to a temp file, then play it with aplay.
	// We can use the existing SoundCacheDir for temp files to keep things clean.
	cacheDir := SoundCacheDir()
	os.MkdirAll(cacheDir, 0755)

	// Create a temporary wav file
	tmpFile, err := os.CreateTemp(cacheDir, "play-*.wav")
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	// We must encode the stream to wav, which requires seeking.
	// The temporary file works perfectly for this.
	err = wav.Encode(tmpFile, streamer, format)
	tmpFile.Close() // Close so aplay can read it

	if cleanup != nil {
		cleanup() // Streamer is fully consumed, cleanup can run
	}

	if err != nil {
		os.Remove(tmpFile.Name())
		return fmt.Errorf("failed to encode stream to wav: %w", err)
	}

	// We'll use aplay which is the standard ALSA audio player on Linux.
	// We run it as a subprocess to play the wav file.
	cmd := exec.Command("aplay", "-q", tmpFile.Name())

	if blocking {
		err := cmd.Run()
		os.Remove(tmpFile.Name())
		if err != nil {
			return fmt.Errorf("aplay failed: %w", err)
		}
	} else {
		// Run in background and cleanup after
		if err := cmd.Start(); err != nil {
			os.Remove(tmpFile.Name())
			return fmt.Errorf("failed to start aplay (is it installed?): %w", err)
		}

		go func() {
			cmd.Wait()
			os.Remove(tmpFile.Name())
		}()
	}

	return nil
}
