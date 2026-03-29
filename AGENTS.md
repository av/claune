# Claune

## Build & Test

After any code changes, always rebuild and reinstall the binary so it can be tested locally:

```
PATH="/home/everlier/go/bin:$PATH" go test ./... && PATH="/home/everlier/go/bin:$PATH" make install
```

The Go binary is at `/home/everlier/go/bin/go`. The install target puts the binary in `~/.local/bin/claune`.
