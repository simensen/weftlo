.PHONY: all build test fmt lint vet clean check install uninstall

# Binary name
BINARY_NAME=weftlo

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOFMT=gofmt
GOVET=$(GOCMD) vet

# Build directory
BUILD_DIR=bin

# Installation directories (user-overridable)
PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin
DESTDIR ?=

all: check build

build:
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/weftlo

test:
	$(GOTEST) -v ./...

test-short:
	$(GOTEST) -short ./...

test-coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

fmt:
	$(GOFMT) -w .

fmt-check:
	@test -z "$$($(GOFMT) -l .)" || (echo "Files need formatting:"; $(GOFMT) -l .; exit 1)

vet:
	$(GOVET) ./...

lint:
	golangci-lint run

check: fmt-check vet lint

clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

install: build
	@mkdir -p $(DESTDIR)$(BINDIR)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(DESTDIR)$(BINDIR)/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to $(DESTDIR)$(BINDIR)/$(BINARY_NAME)"
	@if [ -z "$(DESTDIR)" ] && ! echo "$$PATH" | tr ':' '\n' | grep -qx "$(BINDIR)"; then \
		echo ""; \
		echo "Warning: $(BINDIR) is not in your PATH"; \
		echo "Add it with: export PATH=\"$(BINDIR):\$$PATH\""; \
	fi

uninstall:
	@rm -f $(DESTDIR)$(BINDIR)/$(BINARY_NAME)
	@echo "Removed $(BINARY_NAME) from $(DESTDIR)$(BINDIR)"
