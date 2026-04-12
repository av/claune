package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackCmd_UnknownPack(t *testing.T) {
	err := handlePack([]string{"claune", "pack", "unknown-pack"})
	if err == nil {
		t.Fatalf("expected error for unknown pack, got nil")
	}
	if !strings.Contains(err.Error(), "unknown sound pack 'unknown-pack'") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestPackCmd_InvalidLocalJSON(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	
	invalidJsonPath := filepath.Join(home, "invalid.json")
	if err := os.WriteFile(invalidJsonPath, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	err := handlePack([]string{"claune", "pack", invalidJsonPath})
	if err == nil {
		t.Fatalf("expected error for invalid json, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse custom pack JSON") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestPackCmd_CustomURL(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name":"custom-url-pack","description":"test","sounds":{"cli:start":"test-sound"}}`))
	}))
	defer ts.Close()

	err := handlePack([]string{"claune", "pack", ts.URL})
	if err != nil {
		t.Fatalf("expected nil for valid custom URL, got: %v", err)
	}

	// Verify that the custom URL name was logged or properly handled.
	// We'd expect the config to have been updated to include "cli:start".
	// Though we mock the url, circus.ImportMemeSound will fail to download test-sound since we don't mock it here,
	// but handlePack should still return nil because it continues on error during download.
}

func TestPackCmd_CustomURLInvalidJSON(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json format`))
	}))
	defer ts.Close()

	err := handlePack([]string{"claune", "pack", ts.URL})
	if err == nil {
		t.Fatalf("expected error for invalid custom URL JSON, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse custom pack JSON") {
		t.Errorf("unexpected error message: %v", err)
	}
}
