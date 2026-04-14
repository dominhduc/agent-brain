.PHONY: build install install-local test clean release setup

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

build:
	go build -buildvcs=false -ldflags "$(LDFLAGS)" -o bin/brain ./cmd/brain

install: build
	cp bin/brain /usr/local/bin/brain

install-local: build
	mkdir -p ~/.local/bin
	cp bin/brain ~/.local/bin/brain
	chmod +x ~/.local/bin/brain
	@echo "Installed to ~/.local/bin/brain"
	@echo "Make sure ~/.local/bin is in your PATH"

setup: install-local
	@if ! grep -q '.local/bin' ~/.bashrc 2>/dev/null; then \
		echo 'export PATH="$$HOME/.local/bin:$$PATH"' >> ~/.bashrc; \
		echo "Added ~/.local/bin to ~/.bashrc"; \
	fi
	@echo "Setup complete! Run: brain --help"

test:
	go test ./...

clean:
	rm -rf bin/

release:
	goreleaser release --clean
