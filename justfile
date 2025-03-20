# MegaTool justfile

# Default recipe to run when just is called without arguments
default:
    @just --list

# Initialize the project (install dependencies)
init:
    mise exec -- go mod tidy

# Build all binaries
build:
    mise exec -- go build -o bin/megatool ./cmd/megatool
    mise exec -- go build -o bin/megatool-calculator ./cmd/megatool-calculator
    mise exec -- go build -o bin/megatool-github ./cmd/megatool-github
    mise exec -- go build -o bin/megatool-package-version ./cmd/megatool-package-version
    @echo "Binaries built in ./bin directory"

# Run tests with coverage
test:
    mise exec -- go test -coverprofile=coverage.out ./...
    mise exec -- go tool cover -func=coverage.out

# Run code quality checks
lint:
    mise exec -- go fmt ./...
    mise exec -- go vet ./...
    @echo "Skipping staticcheck (not installed)"
    # mise exec -- staticcheck ./...

# Install binaries
install:
    mise exec -- go install ./cmd/megatool
    mise exec -- go install ./cmd/megatool-calculator
    mise exec -- go install ./cmd/megatool-github
    mise exec -- go install ./cmd/megatool-package-version
    @echo "Binaries installed"

# Run the calculator MCP server (for development)
run-calculator:
    mise exec -- go run ./cmd/megatool-calculator

# Run the GitHub MCP server (for development)
run-github:
    mise exec -- go run ./cmd/megatool-github

# Run the Package Version MCP server (for development)
run-package-version:
    mise exec -- go run ./cmd/megatool-package-version

# Configure the GitHub MCP server
configure-github:
    mise exec -- go run ./cmd/megatool-github --configure

# Clean build artifacts
clean:
    rm -rf bin
    rm -f coverage.out
