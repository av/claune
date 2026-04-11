package cli

import (
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
