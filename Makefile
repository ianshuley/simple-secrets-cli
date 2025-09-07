BINARY=simple-secrets
PREFIX?=/usr/local

# Version information
VERSION?=v0.1.0
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION=$(shell go version | cut -d' ' -f3)

# Build flags for version injection
VERSION_PKG=simple-secrets/pkg/version
LDFLAGS=-ldflags "\
	-X '$(VERSION_PKG).Version=$(VERSION)' \
	-X '$(VERSION_PKG).GitCommit=$(GIT_COMMIT)' \
	-X '$(VERSION_PKG).BuildDate=$(BUILD_DATE)' \
"

all: build

build:
	go build $(LDFLAGS) -o $(BINARY) .

install: build
	mkdir -p $(PREFIX)/bin
	install -m 0755 $(BINARY) $(PREFIX)/bin/$(BINARY)
	@echo ""
	@echo "$(BINARY) installed to $(PREFIX)/bin/$(BINARY)"
	@echo ""
	@echo "To enable shell completion, run one of:"
	@echo "  # For current session only:"
	@echo "  source <($(BINARY) completion zsh)   # zsh"
	@echo "  source <($(BINARY) completion bash)  # bash"
	@echo ""
	@echo "  # For permanent setup:"
	@echo "  $(BINARY) completion zsh --help   # zsh instructions"
	@echo "  $(BINARY) completion bash --help  # bash instructions"

uninstall:
	rm -f $(PREFIX)/bin/$(BINARY)

clean:
	rm -f $(BINARY)

# Development build with version info
dev: VERSION=dev-$(GIT_COMMIT)
dev: build

# Release build with specific version
release: build

test:
	go test -v ./...

integration-test: build
	go test -v ./integration

help:
	@echo "Available targets:"
	@echo "  build           - Build the binary with version info"
	@echo "  dev             - Build development version"
	@echo "  release         - Build release version (set VERSION=vX.Y.Z)"
	@echo "  test            - Run all tests"
	@echo "  integration-test - Run integration tests only"
	@echo "  install         - Install to $(PREFIX)/bin"
	@echo "  uninstall       - Remove from system"
	@echo "  clean           - Remove built binary"
	@echo "  help            - Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION         - Version string (default: $(VERSION))"
	@echo "  PREFIX          - Installation prefix (default: /usr/local)"
	@echo ""
	@echo "Examples:"
	@echo "  make dev                           # Build development version"
	@echo "  make release VERSION=v1.0.0       # Build specific version"
	@echo "  make install PREFIX=$$HOME/.local  # Install to home directory"
	@echo "  sudo make install                  # Install system-wide"
