PREFIX ?= $(HOME)/.local

.PHONY: build test install uninstall clean

build:
	go build -o claune ./cmd/claune

test:
	go test ./...

install: build
	mkdir -p $(PREFIX)/bin
	cp claune $(PREFIX)/bin/claune
	@echo ""
	@echo "Installed to $(PREFIX)/bin/claune"
	@echo "Run 'claune install' to activate sound hooks in Claude Code."

uninstall:
	rm -f $(PREFIX)/bin/claune
	@echo "Removed $(PREFIX)/bin/claune"

clean:
	rm -f claune
