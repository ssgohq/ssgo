# ssgo Makefile
# Build and development commands

.PHONY: build test lint install clean fmt vet tidy help

# Binary name and paths
BINARY_NAME := ss
CMD_PATH := ./tool/cmd/ssgo
BIN_DIR := bin

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS := -s -w \
	-X main.Version=$(VERSION) \
	-X main.GitCommit=$(GIT_COMMIT) \
	-X main.BuildDate=$(BUILD_DATE)

# Default target
.DEFAULT_GOAL := help

## build: Build the CLI binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "Built $(BIN_DIR)/$(BINARY_NAME)"

## test: Run all tests
test:
	@echo "Running tests..."
	go test -race -cover ./...

## test-verbose: Run all tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	go test -race -cover -v ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## lint: Run golangci-lint
lint:
	@echo "Running linter..."
	golangci-lint run ./...

## lint-fix: Run golangci-lint with auto-fix
lint-fix:
	@echo "Running linter with auto-fix..."
	golangci-lint run --fix ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	gofumpt -l -w .
	goimports -local github.com/ssgohq/ssgo -w .

## vet: Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

## tidy: Tidy go modules
tidy:
	@echo "Tidying modules..."
	go mod tidy

## install: Install CLI locally
install:
	@echo "Installing $(BINARY_NAME)..."
	go install -ldflags "$(LDFLAGS)" $(CMD_PATH)
	@echo "Installed $(BINARY_NAME) to $(shell go env GOPATH)/bin"

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html
	@echo "Cleaned"

## snapshot: Build snapshot release using goreleaser
snapshot:
	@echo "Building snapshot..."
	goreleaser release --snapshot --clean

## release-dry-run: Dry run release
release-dry-run:
	@echo "Running release dry-run..."
	goreleaser release --skip=publish --clean

## all: Run fmt, lint, test, and build
all: fmt lint test build

## ci: Run all CI checks (lint, test)
ci: lint test

## help: Show this help message
help:
	@echo "ssgo - All-in-one Go development toolkit"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'