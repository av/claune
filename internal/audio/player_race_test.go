package audio

import (
	"os"
	"sync"
	"testing"
)

func TestPickSoundRoundRobinRace(t *testing.T) {
	// Ensure we start clean
	os.Remove(stateFilePath())
	
	// Create multiple goroutines that concurrently pick sound
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pickSound("test:event:race", []string{"a", "b", "c", "d", "e"}, "round_robin")
		}()
	}
	wg.Wait()
	
	// Load the final state
	loadState()
	idx := rrIndex["test:event:race"]
	
	// It should ideally be 100 % 5 = 0
	t.Logf("Final index: %d", idx)
	if idx != 0 {
		t.Errorf("Expected index 0, got %d", idx)
	}
}
