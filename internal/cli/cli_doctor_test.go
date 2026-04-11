package cli

import (
	"testing"
)

func TestDoctorCmd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	
	output := captureOutput(t, func() {
		if err := Run([]string{"doctor"}, "test-version"); err != nil {
			t.Fatalf("Run(doctor) error = %v", err)
		}
	})

	assertContains(t, output.stdout, "Claune Doctor")
	assertContains(t, output.stdout, "OS:")
	assertContains(t, output.stdout, "Audio Dependencies:")
}
