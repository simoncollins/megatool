# MegaTool justfile

# Run command with mise if available, otherwise run directly
_run cmd="echo":
    @sh -c "if command -v mise >/dev/null 2>&1; then mise exec -- {{cmd}}; else {{cmd}}; fi"

# Default recipe to run when just is called without arguments
default:
    just --list

# Initialize the project (install dependencies)
init:
    @just _run "go mod tidy"

# Build all binaries
build:
    @just _run "go build -o bin/megatool ./cmd/megatool"
    @just _run "go build -o bin/megatool-calculator ./cmd/megatool-calculator"
    @just _run "go build -o bin/megatool-github ./cmd/megatool-github"
    @just _run "go build -o bin/megatool-package-version ./cmd/megatool-package-version"
    @just _run "go build -o bin/megatool-example ./cmd/megatool-example"
    @echo "Binaries built in ./bin directory"

# Run tests and generate coverage report
test:
    @just _run "go test -coverprofile=coverage.out ./..."
    @just _run "go tool cover -func=coverage.out"
    @just _run "go test -v ./internal/mcpserver -run TestSSEServerCompliance"

# Run code quality checks
lint:
    @just _run "go fmt ./..."
    @just _run "go vet ./..."
    @echo "Skipping staticcheck (not installed)"
    # @just _run "staticcheck ./..."

# Install binaries
install:
    @just _run "go install ./cmd/megatool"
    @just _run "go install ./cmd/megatool-calculator"
    @just _run "go install ./cmd/megatool-github"
    @just _run "go install ./cmd/megatool-package-version"
    @just _run "go install ./cmd/megatool-example"
    @echo "Binaries installed"

# Run the calculator MCP server (for development)
run-calculator:
    @just _run "go run ./cmd/megatool-calculator"

# Run the GitHub MCP server (for development)
run-github:
    @just _run "go run ./cmd/megatool-github"

# Run the Package Version MCP server (for development)
run-package-version:
    @just _run "go run ./cmd/megatool-package-version"

# Run the Example MCP server (for development)
run-example:
    @just _run "go run ./cmd/megatool-example"

# Configure the GitHub MCP server
configure-github:
    @just _run "go run ./cmd/megatool-github --configure"

# Clean build artifacts
clean:
    @just _run "rm -rf bin"
    @just _run "rm -f coverage.out"

# Update version and create a matching git tag (prepare for release)
version VERSION:
    @echo "Checking version format..."
    @if echo "{{VERSION}}" | grep -q "^v"; then echo "Error: VERSION should not start with 'v' prefix"; exit 1; fi
    @echo "Updating to version {{VERSION}}"
    @sed -i '' 's/const Version = ".*"/const Version = "{{VERSION}}"/' internal/version/version.go
    @git add internal/version/version.go
    @git commit -m "chore: bump version to {{VERSION}}"
    @git tag -a "v{{VERSION}}" -m "Release v{{VERSION}}"
    @echo "Version updated and tagged. To publish this release, run: just release"

# Push code and tags to trigger a release
release:
    @echo "Pushing code and tags to trigger release workflow..."
    @git push
    @git push --tags
    @VERSION=$$(grep 'const Version =' internal/version/version.go | cut -d'"' -f2) && echo "Release v$$VERSION deployment triggered!"

# List all version tags
version-list:
    @echo "Listing all version tags:"
    @git tag -l "v*" | sort -V
