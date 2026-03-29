//go:build !linux

package audio

import (
	"fmt"
	"time"

	"github.com/gopxl/beep"
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
		// Not implementing complex volume mapping for now to keep dependencies low,
		// but you can add beep/effects.Volume if desired.
		// For simplicity, we just play it as is.
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
