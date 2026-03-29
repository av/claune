package circus

import (
	"strings"

	"github.com/everlier/claune/internal/audio"
	"github.com/everlier/claune/internal/config"
)

// AnalyzeLogSentiment takes terminal output, heuristically analyzes the sentiment or "vibe",
// and triggers the comedically appropriate circus sound via the audio package asynchronously.
func AnalyzeLogSentiment(logText string, c config.ClauneConfig, blocking bool) {
	lowerText := strings.ToLower(logText)

	// Quick AI-like heuristic
	var soundEvent string
	if strings.Contains(lowerText, "panic:") || strings.Contains(lowerText, "fatal") {
		soundEvent = "panic" // maps to "maniacal-laugh.mp3"
	} else if strings.Contains(lowerText, "fail") || strings.Contains(lowerText, "error") {
		soundEvent = "test:fail" // maps to "sad-trombone.mp3"
	} else if strings.Contains(lowerText, "success") || strings.Contains(lowerText, "pass") || strings.Contains(lowerText, "ok") {
		soundEvent = "build:success" // maps to "slide-whistle-up.mp3"
	} else if strings.Contains(lowerText, "warn") {
		soundEvent = "warn" // maps to "boing.mp3"
	} else {
		// Neutral
		soundEvent = "tool:start" // maps to "clown-horn.mp3"
	}

	if blocking {
		_ = audio.PlaySound(soundEvent, true, c)
	} else {
		// Play asynchronously in a goroutine so it's non-blocking execution
		go func() {
			_ = audio.PlaySound(soundEvent, false, c)
		}()
	}
}
