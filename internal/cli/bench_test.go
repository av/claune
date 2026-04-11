package cli

import (
	"bytes"
	"os"
	"testing"
)

func BenchmarkMustReadStdin(b *testing.B) {
	data := bytes.Repeat([]byte("A"), 10*1024*1024) // 10MB
	for i := 0; i < b.N; i++ {
		r, w, _ := os.Pipe()
		oldStdin := os.Stdin
		os.Stdin = r

		go func() {
			w.Write(data)
			w.Close()
		}()

		mustReadStdin("test")
		os.Stdin = oldStdin
		r.Close()
	}
}
