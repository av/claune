//go:build linux

package audio

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
)

func playMP3Stream(streamer beep.StreamSeekCloser, format beep.Format, volume float64, blocking bool, cleanup func()) error {
	cacheDir := SoundCacheDir()
	os.MkdirAll(cacheDir, 0755)

	var ctrl beep.Streamer = streamer

	// Limit playback to 5 minutes max to prevent disk exhaustion
	// from massive uncompressed WAV encodings
	maxFrames := format.SampleRate.N(5 * time.Minute)
	ctrl = beep.Take(maxFrames, ctrl)

	if volume != 1.0 {
		volLog := 0.0
		if volume > 0.001 {
			volLog = math.Log2(volume)
		}
		ctrl = &effects.Volume{
			Streamer: ctrl,
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

		err = safeEncodeWav(tmpFile, ctrl, format)
		syncErr := tmpFile.Sync()
		closeErr := tmpFile.Close()

		if cleanup != nil {
			cleanup()
		}

		if err != nil {
			os.Remove(tmpFile.Name())
			return fmt.Errorf("failed to encode stream to wav: %w", err)
		}
		if syncErr != nil {
			os.Remove(tmpFile.Name())
			return fmt.Errorf("failed to sync wav file to disk: %w", syncErr)
		}
		if closeErr != nil {
			os.Remove(tmpFile.Name())
			return fmt.Errorf("failed to close wav file: %w", closeErr)
		}

		type backend struct {
			bin  string
			args []string
		}

		backends := []backend{
			{"paplay", []string{tmpFile.Name()}},
			{"pw-play", []string{tmpFile.Name()}},
			{"aplay", []string{"-q", tmpFile.Name()}},
		}

		var lastErr error
		played := false

		for _, b := range backends {
			if path, err := exec.LookPath(b.bin); err == nil {
				if b.bin == "aplay" {
					lockPath := filepath.Join(cacheDir, ".aplay.lock")
					locked := false
					for j := 0; j < 500; j++ {
						f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
						if err == nil {
							f.Close()
							locked = true
							defer os.Remove(lockPath)
							break
						}
						if info, err := os.Stat(lockPath); err == nil && time.Since(info.ModTime()) > 30*time.Second {
							os.Remove(lockPath)
							continue
						}
						time.Sleep(10 * time.Millisecond)
					}
					if !locked {
						lastErr = fmt.Errorf("aplay lock timeout")
						continue
					}
				}

				shScript := `
					target_file=$1
					ppid=$2
					shift 2
					"$@" &
					pid=$!
					trap 'kill -TERM $pid 2>/dev/null; rm -f "$target_file"; exit 1' INT TERM
					wait $pid
					exit_code=$?
					if [ $exit_code -eq 0 ] || ! kill -0 $ppid 2>/dev/null; then
						rm -f "$target_file"
					fi
					exit $exit_code
				`
				args := append([]string{"-c", shScript, "--", tmpFile.Name(), fmt.Sprint(os.Getpid()), path}, b.args...)
				
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				cmd := exec.CommandContext(ctx, "sh", args...)
				err := cmd.Run()
				cancel()

				if err == nil {
					played = true
					break
				}
				lastErr = fmt.Errorf("%s failed: %w", b.bin, err)
			}
		}

		os.Remove(tmpFile.Name())

		if !played {
			if lastErr != nil {
				return fmt.Errorf("all audio backends failed (paplay, pw-play, aplay) - missing audio dependency or daemon not running: %w", lastErr)
			}
			return fmt.Errorf("missing audio dependency: please install paplay (pulseaudio/libpulse), pw-play (pipewire), or aplay (alsa)")
		}

		return nil
	}

	if blocking {
		return doPlay()
	}

	go doPlay()
	return nil
}
