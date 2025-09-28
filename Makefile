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
	@$(MAKE) install-completions
	@echo ""
	@echo "Installation complete!"

install-completions:
	@echo ""
	@echo "Installing shell completions..."
	@SHELL_NAME=$$(basename "$$SHELL" 2>/dev/null || echo "unknown"); \
	case "$$SHELL_NAME" in \
		zsh) \
			if [ -d "$$HOME/.oh-my-zsh" ]; then \
				COMP_DIR="$$HOME/.oh-my-zsh/completions"; \
				mkdir -p "$$COMP_DIR"; \
			elif [ -d "/usr/local/share/zsh/site-functions" ] && [ -w "/usr/local/share/zsh/site-functions" ]; then \
				COMP_DIR="/usr/local/share/zsh/site-functions"; \
			elif [ -d "/usr/share/zsh/site-functions" ] && [ -w "/usr/share/zsh/site-functions" ]; then \
				COMP_DIR="/usr/share/zsh/site-functions"; \
			else \
				COMP_DIR="$$HOME/.zsh/completions"; \
				mkdir -p "$$COMP_DIR"; \
			fi; \
			echo "  → Installing zsh completions to $$COMP_DIR/_$(BINARY)"; \
			$(PREFIX)/bin/$(BINARY) completion zsh > "$$COMP_DIR/_$(BINARY)" 2>/dev/null || true; \
			echo "  ✅ zsh completions installed (restart shell or run: exec zsh)"; \
			;; \
		bash) \
			if [ -d "/usr/local/etc/bash_completion.d" ] && [ -w "/usr/local/etc/bash_completion.d" ]; then \
				COMP_DIR="/usr/local/etc/bash_completion.d"; \
			elif [ -d "/etc/bash_completion.d" ] && [ -w "/etc/bash_completion.d" ]; then \
				COMP_DIR="/etc/bash_completion.d"; \
			else \
				COMP_DIR="$$HOME/.bash_completions"; \
				mkdir -p "$$COMP_DIR"; \
			fi; \
			echo "  → Installing bash completions to $$COMP_DIR/$(BINARY)"; \
			$(PREFIX)/bin/$(BINARY) completion bash > "$$COMP_DIR/$(BINARY)" 2>/dev/null || true; \
			if [ "$$COMP_DIR" = "$$HOME/.bash_completions" ]; then \
				echo "  💡 Add to ~/.bashrc: source $$HOME/.bash_completions/$(BINARY)"; \
			fi; \
			echo "  ✅ bash completions installed (restart shell or source ~/.bashrc)"; \
			;; \
		fish) \
			if [ -d "$$HOME/.config/fish/completions" ]; then \
				COMP_DIR="$$HOME/.config/fish/completions"; \
			else \
				COMP_DIR="$$HOME/.config/fish/completions"; \
				mkdir -p "$$COMP_DIR"; \
			fi; \
			echo "  → Installing fish completions to $$COMP_DIR/$(BINARY).fish"; \
			$(PREFIX)/bin/$(BINARY) completion fish > "$$COMP_DIR/$(BINARY).fish" 2>/dev/null || true; \
			echo "  ✅ fish completions installed (restart shell)"; \
			;; \
		*) \
			echo "  ⚠️  Shell '$$SHELL_NAME' not recognized for auto-completion"; \
			echo "  💡 Manual setup available:"; \
			echo "     $(BINARY) completion zsh --help   # zsh instructions"; \
			echo "     $(BINARY) completion bash --help  # bash instructions"; \
			echo "     $(BINARY) completion fish --help  # fish instructions"; \
			;; \
	esac

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
	@echo "  build              - Build the binary with version info"
	@echo "  dev                - Build development version"
	@echo "  release            - Build release version (set VERSION=vX.Y.Z)"
	@echo "  test               - Run all tests"
	@echo "  integration-test   - Run integration tests only"
	@echo "  install            - Install binary + auto-detect shell completions"
	@echo "  install-completions - Install shell completions only"
	@echo "  uninstall          - Remove from system"
	@echo "  clean              - Remove built binary"
	@echo "  help               - Show this help message"
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
