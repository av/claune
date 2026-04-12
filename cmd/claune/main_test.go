package main

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"testing"
)

func TestMainHelpSuccess(t *testing.T) {
	oldArgs := os.Args
	oldVersion := Version
	oldRunCLI := runCLI
	oldStderr := stderr
	oldExit := exit
	defer func() {
		os.Args = oldArgs
		Version = oldVersion
		runCLI = oldRunCLI
		stderr = oldStderr
		exit = oldExit
	}()

	os.Args = []string{"claune", "--help"}
	Version = "test-version"

	var errBuf bytes.Buffer
	stderr = &errBuf

	var gotArgs []string
	var gotVersion string
	var exitCalled bool
	runCLI = func(args []string, version string) error {
		gotArgs = append([]string(nil), args...)
		gotVersion = version
		return nil
	}
	exit = func(code int) {
		exitCalled = true
		t.Fatalf("unexpected exit(%d) on success", code)
	}

	main()

	if !reflect.DeepEqual(gotArgs, []string{"--help"}) {
		t.Fatalf("main() delegated args = %#v, want %#v", gotArgs, []string{"--help"})
	}

	if gotVersion != "test-version" {
		t.Fatalf("main() delegated version = %q, want %q", gotVersion, "test-version")
	}

	if exitCalled {
		t.Fatal("main() should not exit on successful help path")
	}

	if gotStderr := errBuf.String(); gotStderr != "" {
		t.Fatalf("main() wrote unexpected stderr on success: %q", gotStderr)
	}
}

func TestMainError(t *testing.T) {
	oldArgs := os.Args
	oldVersion := Version
	oldRunCLI := runCLI
	oldStderr := stderr
	oldExit := exit
	defer func() {
		os.Args = oldArgs
		Version = oldVersion
		runCLI = oldRunCLI
		stderr = oldStderr
		exit = oldExit
	}()

	os.Args = []string{"claune", "version"}
	Version = "test-version"

	var errBuf bytes.Buffer
	stderr = &errBuf

	var exitCode int
	var exitCalled bool
	runCLI = func(args []string, version string) error {
		return errors.New("boom")
	}
	exit = func(code int) {
		exitCalled = true
		exitCode = code
	}

	main()

	if !exitCalled {
		t.Fatal("main() did not exit on error")
	}

	if exitCode != 1 {
		t.Fatalf("main() exit code = %d, want 1", exitCode)
	}

	if gotStderr := errBuf.String(); gotStderr != "Error: boom\n" {
		t.Fatalf("main() wrote stderr = %q, want %q", gotStderr, "Error: boom\n")
	}
}
