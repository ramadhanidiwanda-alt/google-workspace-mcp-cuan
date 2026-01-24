# Makefile for Google Workspace MCP Server (Go)

BINARY_NAME=workspace-server
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -s -w"

.PHONY: all build test clean release install

all: build

# Build for current platform
build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/workspace-server

# Run the server
run: build
	./bin/$(BINARY_NAME)

# Run with debug logging
run-debug: build
	./bin/$(BINARY_NAME) --debug

# All unit tests
test:
	go test -v ./internal/...

# Unit tests (short mode)
test-unit:
	go test -short -v ./internal/...

# Integration tests
test-integration:
	go test -v ./test/integration/...

# E2E tests (requires authentication)
test-e2e:
	GOOGLE_WORKSPACE_TEST=1 go test -v ./test/e2e/...

# Test coverage
test-coverage:
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# All tests (excluding E2E)
test-all: test test-integration
	@echo "All tests passed"

# CI tests with race detection
test-ci:
	go test -race -coverprofile=coverage.out ./internal/... ./test/integration/...

# Lint
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -rf bin/ dist/ coverage.out coverage.html

# Download dependencies
deps:
	go mod download
	go mod tidy

# Cross-compilation for releases
release: clean
	# macOS ARM64
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/workspace-server
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/workspace-server
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/workspace-server
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/workspace-server
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/workspace-server

# Create release archives
package: release
	cd dist && \
	tar czf $(BINARY_NAME)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64 && \
	tar czf $(BINARY_NAME)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64 && \
	tar czf $(BINARY_NAME)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64 && \
	tar czf $(BINARY_NAME)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64 && \
	zip $(BINARY_NAME)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe

# Install to GOPATH/bin
install: build
	cp bin/$(BINARY_NAME) $(GOPATH)/bin/
