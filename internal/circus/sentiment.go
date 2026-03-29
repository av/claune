package circus

import (
	"strings"

	"github.com/everlier/claune/internal/audio"
	"github.com/everlier/claune/internal/config"
)

// AnalyzeLogSentiment takes terminal output, heuristically analyzes the sentiment or "vibe",
// and triggers the comedically appropriate circus sound via the audio package asynchronously.
func AnalyzeLogSentiment(logText string, c config.ClauneConfig) {
	lowerText := strings.ToLower(logText)

	// Quick AI-like heuristic
	var soundEvent string
	if strings.Contains(lowerText, "panic:") || strings.Contains(lowerText, "fatal") {
		soundEvent = "panic" // maps to "maniacal-laugh.wav"
	} else if strings.Contains(lowerText, "fail") || strings.Contains(lowerText, "error") {
		soundEvent = "test:fail" // maps to "sad-trombone.wav"
	} else if strings.Contains(lowerText, "success") || strings.Contains(lowerText, "pass") || strings.Contains(lowerText, "ok") {
		soundEvent = "build:success" // maps to "slide-whistle-up.wav"
	} else if strings.Contains(lowerText, "warn") {
		soundEvent = "warn" // maps to "boing.wav"
	} else {
		// Neutral
		soundEvent = "tool:start" // maps to "clown-horn.wav"
	}

	// Play asynchronously in a goroutine so it's non-blocking execution
	go func() {
		_ = audio.PlaySound(soundEvent, false, c)
	}()
}
