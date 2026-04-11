package circus

import (
	"testing"
	"github.com/everlier/claune/internal/config"
)

func TestAnalyzeLogSentiment_NonBlockingDoesNotPanic(t *testing.T) {
	c := config.ClauneConfig{}
	
	// Test various branches for sentiment coverage
	tests := []string{
		"panic: something bad happened",
		"fatal error occurred",
		"build fail",
		"error encountered",
		"build success",
		"all tests pass",
		"everything is ok",
		"warn: deprecated",
		"neutral log message",
	}
	
	for _, text := range tests {
		AnalyzeLogSentiment(text, c, false)
	}
}

func TestAnalyzeLogSentiment_BlockingDoesNotPanic(t *testing.T) {
	c := config.ClauneConfig{}
	
	// Test the blocking path
	AnalyzeLogSentiment("panic: something bad happened", c, true)
}
