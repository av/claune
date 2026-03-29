package audio

import "testing"

func TestSoundMap(t *testing.T) {
	if len(DefaultSoundMap) == 0 {
		t.Error("No sounds in map")
	}
}
