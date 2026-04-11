package circus

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestImportMemeSoundExceeds50MBLimit(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Write exactly 50MB + 1 byte
		chunk := bytes.Repeat([]byte("A"), 1024*1024)
		for i := 0; i < 50; i++ {
			w.Write(chunk)
		}
		w.Write([]byte("B"))
	}))
	defer server.Close()

	err := ImportMemeSound(server.URL+"/huge.mp3", "huge.mp3")
	if err == nil {
		t.Fatal("expected error for file exceeding 50MB limit")
	}
	if !strings.Contains(err.Error(), "exceeds 50MB limit") {
		t.Fatalf("expected 50MB limit error, got: %v", err)
	}
}
