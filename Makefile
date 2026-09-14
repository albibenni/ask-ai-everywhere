BINARY := ask-ai
COMMAND := ./cmd/ask-ai
BUILD_DIR := bin
INSTALL_PREFIX ?= $(HOME)/.local
VERSION ?= dev
GO_LDFLAGS := -X main.version=$(VERSION)

.DEFAULT_GOAL := help

.PHONY: help fmt test vet check build install cross-build clean

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*## "; printf "Usage: make <target>\n\nTargets:\n"} /^[a-zA-Z_-]+:.*## / {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

fmt: ## Format all Go source files
	gofmt -w cmd internal

test: ## Run the test suite with the race detector
	go test -race ./...

vet: ## Run Go's static analyzer
	go vet ./...

check: test vet ## Run all verification checks

build: ## Build the executable for the current platform
	mkdir -p $(BUILD_DIR)
	go build -ldflags "$(GO_LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) $(COMMAND)

install: build ## Install under INSTALL_PREFIX (default: ~/.local)
	mkdir -p $(DESTDIR)$(INSTALL_PREFIX)/bin
	install -m 0755 $(BUILD_DIR)/$(BINARY) $(DESTDIR)$(INSTALL_PREFIX)/bin/$(BINARY)

cross-build: ## Build Linux and macOS release binaries
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(GO_LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-amd64 $(COMMAND)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "$(GO_LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-arm64 $(COMMAND)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "$(GO_LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 $(COMMAND)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "$(GO_LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 $(COMMAND)

clean: ## Remove generated binaries
	rm -rf $(BUILD_DIR)
