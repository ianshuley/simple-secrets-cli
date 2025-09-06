BINARY=simple-secrets
PREFIX?=/usr/local

all: build

build:
	go build -o $(BINARY) .

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

test:
	go test -v ./...

integration-test: build
	go test -v ./integration

help:
	@echo "Available targets:"
	@echo "  build           - Build the binary"
	@echo "  test            - Run all tests"
	@echo "  integration-test - Run integration tests only"
	@echo "  install         - Install to $(PREFIX)/bin"
	@echo "  uninstall       - Remove from system"
	@echo "  clean           - Remove built binary"
	@echo "  help            - Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  PREFIX          - Installation prefix (default: /usr/local)"
	@echo ""
	@echo "Examples:"
	@echo "  make install PREFIX=$$HOME/.local  # Install to home directory"
	@echo "  sudo make install                  # Install system-wide"
