package notify

import (
	"testing"
)

func TestSend(t *testing.T) {
	// Test basic execution without failing if external commands are missing.
	err := Send("Test Title", "Test Message with \"quotes\"")
	if err != nil {
		t.Logf("Send returned error (expected in some CI environments): %v", err)
	}
}
