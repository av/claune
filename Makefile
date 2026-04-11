PREFIX ?= $(HOME)/.local
VERSION ?= $(shell git describe --tags --always --dirty || echo "dev")
LDFLAGS = -ldflags "-X main.Version=$(VERSION)"

.PHONY: build test lint release install uninstall clean

build:
	go build $(LDFLAGS) -o claune ./cmd/claune

test:
	go test ./...

lint:
	go vet ./...
	if command -v golangci-lint > /dev/null; then golangci-lint run; else echo "golangci-lint not installed, skipping"; fi

release:
	@echo "Building release binaries..."
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/claune-linux-amd64 ./cmd/claune
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/claune-linux-arm64 ./cmd/claune
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/claune-darwin-amd64 ./cmd/claune
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/claune-darwin-arm64 ./cmd/claune
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/claune-windows-amd64.exe ./cmd/claune
	@echo "Release binaries built in dist/ directory"

install: build
	mkdir -p $(PREFIX)/bin
	rm -f $(PREFIX)/bin/claune
	cp claune $(PREFIX)/bin/claune
	chmod 755 $(PREFIX)/bin/claune
	@echo ""
	@echo "Installed to $(PREFIX)/bin/claune"
	@echo "Run 'claune install' to activate sound hooks in Claude Code."

uninstall:
	rm -f $(PREFIX)/bin/claune
	@echo "Removed $(PREFIX)/bin/claune"

clean:
	rm -f claune
	rm -rf dist
