package cli

import (
	"os"
	"testing"
)

func TestSetupCmd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create a mock stdin with inputs: "n" (mute), "y" (enable AI), "test-key" (API key), "0.5" (volume)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	_, err = w.Write([]byte("n\ny\ntest-key\n0.5\n"))
	if err != nil {
		t.Fatal(err)
	}
	w.Close()

	// Replace os.Stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()
	os.Stdin = r

	output := captureOutput(t, func() {
		if err := Run([]string{"setup"}, "test-version"); err != nil {
			t.Fatalf("Run(setup) error = %v", err)
		}
	})

	assertContains(t, output.stdout, "Sounds will be muted initially.")
	assertContains(t, output.stdout, "AI features enabled!")
	assertContains(t, output.stdout, "Volume set to 0.5")
	assertContains(t, output.stdout, "Configuration saved to")
}
