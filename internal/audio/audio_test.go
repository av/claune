package audio

import (
	"os"
	"testing"
)

func TestSoundMap(t *testing.T) {
	if len(DefaultSoundMap) == 0 {
		t.Error("No sounds in map")
	}
}

func TestPickSoundRoundRobin(t *testing.T) {
	// Ensure clean state
	os.Remove(stateFilePath())

	sounds := []string{"1.mp3", "2.mp3", "3.mp3"}
	
	// Reset index
	rrMutex.Lock()
	rrIndex["test:event"] = 0
	rrMutex.Unlock()

	expected := []string{"1.mp3", "2.mp3", "3.mp3", "1.mp3", "2.mp3"}
	
	for i, exp := range expected {
		got := pickSound("test:event", sounds, "round_robin")
		if got != exp {
			t.Errorf("Iteration %d: expected %s, got %s", i, exp, got)
		}
	}
}

func TestPickSoundRandom(t *testing.T) {
	sounds := []string{"1.mp3"}
	got := pickSound("test:event", sounds, "random")
	if got != "1.mp3" {
		t.Errorf("Expected 1.mp3, got %s", got)
	}
}
