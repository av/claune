package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestCircularBufferTruncation(t *testing.T) {
	chunkSize := 32 * 1024

	tests := []struct {
		name       string
		size       int
		expectTrun bool
	}{
		{"Exactly 32KB", chunkSize, false},
		{"32KB + 1", chunkSize + 1, false},
		{"Exactly 64KB", chunkSize * 2, false},
		{"64KB + 1", chunkSize*2 + 1, true},
		{"100KB", 100 * 1024, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			oldStdin := os.Stdin
			os.Stdin = r
			defer func() {
				os.Stdin = oldStdin
				r.Close()
			}()

			go func() {
				data := bytes.Repeat([]byte("A"), tc.size)
				w.Write(data)
				w.Close()
			}()

			res := mustReadStdin("test")
			hasTrun := strings.Contains(res, "truncated mid-stream")
			if hasTrun != tc.expectTrun {
				t.Errorf("Expected truncation=%v for size %d, got %v", tc.expectTrun, tc.size, hasTrun)
			}

			// verify size
			expectedLen := tc.size
			if tc.expectTrun {
				expectedLen = chunkSize*2 + len("\n\n... [truncated mid-stream] ...\n\n")
			}
			if len(res) != expectedLen {
				t.Errorf("Expected len=%d, got %d", expectedLen, len(res))
			}
		})
	}
}
