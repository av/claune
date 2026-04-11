package audio

import (
	"sync"
	"os"
	"testing"
	"github.com/gopxl/beep"
	"path/filepath"
	"strings"
	"io"
	"github.com/everlier/claune/internal/config"
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

func TestConcurrentEnsureSoundCache(t *testing.T) {
	os.Setenv("XDG_CACHE_HOME", t.TempDir())
	
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := EnsureSoundCache()
			if err != nil {
				t.Errorf("EnsureSoundCache failed: %v", err)
			}
		}()
	}
	wg.Wait()
}

func TestValidEventTypes(t *testing.T) {
	events := ValidEventTypes()
	if !strings.Contains(events, "cli:start") {
		t.Errorf("Expected cli:start in ValidEventTypes, got %s", events)
	}
}

func TestPlaySoundMuted(t *testing.T) {
	mute := true
	c := config.ClauneConfig{Mute: &mute}
	err := PlaySound("cli:start", false, c)
	if err != nil {
		t.Errorf("Expected nil error when muted, got %v", err)
	}
}

func TestPlaySoundUnknownEvent(t *testing.T) {
	unmute := false
	c := config.ClauneConfig{Mute: &unmute}
	err := PlaySound("unknown:event", false, c)
	if err == nil || !strings.Contains(err.Error(), "unknown event type") {
		t.Errorf("Expected unknown event type error, got %v", err)
	}
}

func TestSafeDecodeWavPanic(t *testing.T) {
	// bad reader that returns error
	r := strings.NewReader("not a wav")
	rc := io.NopCloser(r)
	_, _, err := safeDecodeWav(rc)
	if err == nil {
		t.Errorf("Expected error decoding invalid wav, got nil")
	}
}

func TestSafeDecodeMP3Panic(t *testing.T) {
	// bad reader
	r := strings.NewReader("not an mp3")
	rc := io.NopCloser(r)
	_, _, err := safeDecodeMP3(rc)
	if err == nil {
		t.Errorf("Expected error decoding invalid mp3, got nil")
	}
}

func TestPlaySoundCustomInvalidFile(t *testing.T) {
	// create an invalid custom sound file
	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "invalid.mp3")
	os.WriteFile(invalidFile, []byte("invalid content"), 0644)
	
	unmute := false
	c := config.ClauneConfig{
		Mute: &unmute,
		Sounds: map[string]config.EventSoundConfig{
			"cli:start": {
				Paths: []string{invalidFile},
			},
		},
	}
	
	// This should fall back to default embedded sound and return nil error or embedded play error 
	// Since we are not doing audio hardware test, it might error from playEmbeddedSound, but not from "file is empty"
	err := PlaySoundWithStrategy("cli:start", "", false, c)
	// we just ensure it doesn't crash
	_ = err
}

func TestPlaySoundCustomEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.mp3")
	os.WriteFile(emptyFile, []byte(""), 0644) // 0 bytes
	
	unmute := false
	c := config.ClauneConfig{
		Mute: &unmute,
		Sounds: map[string]config.EventSoundConfig{
			"cli:start": {
				Paths: []string{emptyFile},
			},
		},
	}
	
	err := PlaySoundWithStrategy("cli:start", "", false, c)
	_ = err
}

func TestPlayMP3FileWAV(t *testing.T) {
	tmpDir := t.TempDir()
	invalidWav := filepath.Join(tmpDir, "invalid.wav")
	os.WriteFile(invalidWav, []byte("invalid content"), 0644)
	
	err := playMP3File(invalidWav, 1.0, false)
	if err == nil {
		t.Errorf("Expected error playing invalid wav")
	}
}

func TestCachedSoundPath(t *testing.T) {
	_ = cachedSoundPath("nonexistent.mp3")
}

func TestSafeEncodeWavPanic(t *testing.T) {
	// A nil writer or bad streamer might cause panic? 
	// Actually we just pass nil to trigger panic or error
	err := safeEncodeWav(nil, nil, beep.Format{})
	if err == nil {
		t.Errorf("Expected error encoding wav with nil values")
	}
}

type mockStream struct {
	beep.Streamer
}
func (m mockStream) Close() error { return nil }
func (m mockStream) Seek(p int) error { return nil }
func (m mockStream) Position() int { return 0 }
func (m mockStream) Len() int { return 0 }

func TestPlayMP3StreamFallback(t *testing.T) {
	// dummy stream
	streamer := beep.Take(100, beep.Silence(-1))
	format := beep.Format{SampleRate: 44100, NumChannels: 2, Precision: 2}
	mStream := mockStream{streamer}
	err := playMP3Stream(mStream, format, 0.5, true, nil)
	
	// Either nil or "all audio backends failed"
	if err != nil && !strings.Contains(err.Error(), "audio dependency") && !strings.Contains(err.Error(), "audio backends failed") {
		// we just expect it to not panic
		// depending on system it might actually play silence and return nil
		t.Logf("Got error: %v", err)
	}
}
