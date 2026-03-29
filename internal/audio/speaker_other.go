//go:build !linux

package audio

import (
	"fmt"
	"math"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/speaker"
)

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

func playMP3Stream(streamer beep.StreamSeekCloser, format beep.Format, volume float64, blocking bool, cleanup func()) error {
	err := initSpeaker(format.SampleRate)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return fmt.Errorf("audio unavailable: %w", err)
	}

	done := make(chan bool, 1)

	// Apply volume if needed
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

	seq := beep.Seq(ctrl, beep.Callback(func() {
		if cleanup != nil {
			cleanup()
		}
		done <- true
	}))

	speaker.Play(seq)

	if blocking {
		<-done
	}
	return nil
}
