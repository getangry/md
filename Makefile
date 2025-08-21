.PHONY: build test clean install lint fmt vet coverage help build-all build-linux build-windows build-darwin

# Default target
all: build

# Build the binary
build:
	@echo "Building md..."
	@mkdir -p build
	@go build -o build/md

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Vet code for issues
vet:
	@echo "Vetting code..."
	@go vet ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1; }
	@golangci-lint run

# Install the binary to GOPATH/bin
install: build
	@echo "Installing md..."
	@cp build/md $(GOPATH)/bin/md
	@echo "md installed to $(GOPATH)/bin/md"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf build/ dist/ coverage.out coverage.html

# Run all quality checks
check: fmt vet test
	@echo "All checks passed!"

# Development build with race detection
dev:
	@echo "Building with race detection..."
	@mkdir -p build
	@go build -race -o build/md

# Build matrix for all platforms and architectures
build-all: build-linux build-windows build-darwin
	@echo "All platform builds completed!"

# Build for Linux (amd64 and arm64)
build-linux:
	@echo "Building for Linux..."
	@mkdir -p dist/linux
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o dist/linux/md-linux-amd64
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-w -s" -o dist/linux/md-linux-arm64
	@echo "Linux builds completed"

# Build for Windows (amd64 and arm64)
build-windows:
	@echo "Building for Windows..."
	@mkdir -p dist/windows
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-w -s" -o dist/windows/md-windows-amd64.exe
	@CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags="-w -s" -o dist/windows/md-windows-arm64.exe
	@echo "Windows builds completed"

# Build for macOS (amd64 and arm64)
build-darwin:
	@echo "Building for macOS..."
	@mkdir -p dist/darwin
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-w -s" -o dist/darwin/md-darwin-amd64
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-w -s" -o dist/darwin/md-darwin-arm64
	@echo "macOS builds completed"

# Create release archives
release: build-all
	@echo "Creating release archives..."
	@mkdir -p dist/archives
	@cd dist/linux && tar -czf ../archives/md-linux-amd64.tar.gz md-linux-amd64
	@cd dist/linux && tar -czf ../archives/md-linux-arm64.tar.gz md-linux-arm64
	@cd dist/windows && zip ../archives/md-windows-amd64.zip md-windows-amd64.exe
	@cd dist/windows && zip ../archives/md-windows-arm64.zip md-windows-arm64.exe
	@cd dist/darwin && tar -czf ../archives/md-darwin-amd64.tar.gz md-darwin-amd64
	@cd dist/darwin && tar -czf ../archives/md-darwin-arm64.tar.gz md-darwin-arm64
	@echo "Release archives created in dist/archives/"
	@ls -la dist/archives/

# Show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  build         - Build the binary for current platform"
	@echo "  build-all     - Build for all platforms (Linux, Windows, macOS)"
	@echo "  build-linux   - Build for Linux (amd64 and arm64)"
	@echo "  build-windows - Build for Windows (amd64 and arm64)"
	@echo "  build-darwin  - Build for macOS (amd64 and arm64)"
	@echo "  release       - Create release archives for all platforms"
	@echo ""
	@echo "Development targets:"
	@echo "  test          - Run tests"
	@echo "  coverage      - Run tests with coverage report"
	@echo "  fmt           - Format code"
	@echo "  vet           - Vet code for issues"
	@echo "  lint          - Lint code (requires golangci-lint)"
	@echo "  check         - Run fmt, vet, and test"
	@echo "  dev           - Build with race detection"
	@echo ""
	@echo "Utility targets:"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  clean         - Clean build artifacts"
	@echo "  help          - Show this help message"