# Binary name — change this one line to rename all produced artifacts.
BINARY := pd-cli
DIST   := dist

.PHONY: help dev build build-pi-zero test clean

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  make %-18s %s\n", $$1, $$2}'

dev: ## Run CLI locally (TUI; honors DASHBOARD_API_URL and REFRESH_INTERVAL env vars)
	go run ./cmd

build: ## Build native binary for the current host ($(DIST)/$(BINARY))
	@mkdir -p $(DIST)
	go build -o $(DIST)/$(BINARY) ./cmd

build-pi-zero: ## Cross-compile for Raspberry Pi Zero W (linux/arm/armv6 -> $(DIST)/$(BINARY)-pi-zero)
	@mkdir -p $(DIST)
	GOOS=linux GOARCH=arm GOARM=6 go build -o $(DIST)/$(BINARY)-pi-zero ./cmd

test: ## Run tests
	go test ./...

clean: ## Remove build artifacts
	rm -rf $(DIST)
