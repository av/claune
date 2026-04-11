package audio
import (
    "bytes"
    "testing"
    "io"
)
func FuzzWavDecode(f *testing.F) {
    f.Add([]byte("RIFF    WAVEfmt "))
    f.Fuzz(func(t *testing.T, data []byte) {
        rc := struct {
            io.Reader
            io.Closer
        }{bytes.NewReader(data), io.NopCloser(bytes.NewReader(data))}
        streamer, _, _ := safeDecodeWav(rc)
        if streamer != nil {
            streamer.Close()
        }
    })
}
func FuzzMP3Decode(f *testing.F) {
    f.Add([]byte("ID3"))
    f.Fuzz(func(t *testing.T, data []byte) {
        rc := struct {
            io.Reader
            io.Closer
        }{bytes.NewReader(data), io.NopCloser(bytes.NewReader(data))}
        streamer, _, _ := safeDecodeMP3(rc)
        if streamer != nil {
            streamer.Close()
        }
    })
}
