package circus

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/everlier/claune/internal/audio"
)

func TestImportMemeSoundRejectsInvalidURLScheme(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	err := ImportMemeSound("ftp://example.com/sound.mp3", "sound.mp3")
	if err == nil {
		t.Fatal("expected invalid scheme error")
	}
	if !strings.Contains(err.Error(), "Only http:// and https:// URLs are supported") {
		t.Fatalf("expected invalid scheme error, got %v", err)
	}
}

func TestImportMemeSoundReturnsErrorForNon200Response(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer server.Close()

	err := ImportMemeSound(server.URL+"/sound.mp3", "sound.mp3")
	if err == nil {
		t.Fatal("expected non-200 response error")
	}
	if !strings.Contains(err.Error(), "502 Bad Gateway") {
		t.Fatalf("expected status in error, got %v", err)
	}

	if _, statErr := os.Stat(filepath.Join(audio.SoundCacheDir(), "sound.mp3")); !os.IsNotExist(statErr) {
		t.Fatalf("expected no cached file to be created, stat err = %v", statErr)
	}
}

func TestImportMemeSoundSavesDownloadedFileInCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mp3-bytes"))
	}))
	defer server.Close()

	stdout := captureStdout(t, func() {
		if err := ImportMemeSound(server.URL+"/sound.mp3", "victory.mp3"); err != nil {
			t.Fatalf("expected successful import, got %v", err)
		}
	})
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty so CLI owns success messaging", stdout)
	}

	dest := filepath.Join(audio.SoundCacheDir(), "victory.mp3")
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("expected cached file at %s: %v", dest, err)
	}
	if string(data) != "mp3-bytes" {
		t.Fatalf("expected downloaded contents to be saved, got %q", string(data))
	}
}

func TestImportMemeSoundRejectsNamesThatEscapeCacheDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mp3-bytes"))
	}))
	defer server.Close()

	err := ImportMemeSound(server.URL+"/sound.mp3", filepath.Join("..", "escape.mp3"))
	if err == nil {
		t.Fatal("expected escaping name to be rejected")
	}
	if !strings.Contains(err.Error(), "invalid import name") {
		t.Fatalf("expected invalid import name error, got %v", err)
	}

	if _, statErr := os.Stat(filepath.Join(audio.SoundCacheDir(), "..", "escape.mp3")); !os.IsNotExist(statErr) {
		t.Fatalf("expected no file outside cache dir, stat err = %v", statErr)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}

	originalStdout := os.Stdout
	os.Stdout = writer
	t.Cleanup(func() {
		os.Stdout = originalStdout
	})

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}

	var buffer bytes.Buffer
	if _, err := buffer.ReadFrom(reader); err != nil {
		t.Fatalf("buffer.ReadFrom() error = %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("reader.Close() error = %v", err)
	}

	return buffer.String()
}
