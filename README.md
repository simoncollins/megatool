# MegaTool

MegaTool is a command-line tool that implements multiple Model Context Protocol (MCP) servers. Each MCP server is accessible via the `run` subcommand (e.g., `megatool run github`, `megatool run calculator`). The tool uses separate binaries for each MCP server and a main binary to dispatch to the correct one based on the server name.

## Features

- **Multiple MCP Servers**: Each server runs as a separate binary (e.g., `megatool-github`, `megatool-calculator`, `megatool-package-version`).
- **Dispatcher Architecture**: The main `megatool` binary dispatches to the appropriate MCP server binary.
- **Secure Configuration**: Configuration for each server is managed via a `--configure` flag, with sensitive data (e.g., Personal Access Tokens) stored securely using the system keyring.
- **MCP Communication**: MCP servers communicate with clients over stdio.

## Installation

### Prerequisites

- Go 1.24 or later
- [Mise](https://github.com/jdx/mise) for managing Go toolchain
- [Just](https://github.com/casey/just) for running tasks

### Building from Source

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/megatool.git
   cd megatool
   ```

2. Build the binaries:
   ```bash
   just build
   ```

3. Install the binaries:
   ```bash
   just install
   ```

## Usage

### Basic Usage

```bash
megatool run <server> [flags]
```

Where `<server>` is one of the available MCP servers:
- `calculator`: Simple calculator MCP server
- `github`: GitHub MCP server
- `package-version`: Package version checker MCP server

### Calculator MCP Server

The Calculator MCP server provides basic arithmetic operations.

```bash
# Run the calculator MCP server
megatool run calculator
```

### GitHub MCP Server

The GitHub MCP server provides information about GitHub repositories and users.

```bash
# Configure the GitHub MCP server (required before first use)
megatool run github --configure

# Run the GitHub MCP server
megatool run github
```

### Package Version MCP Server

The Package Version MCP server checks for the latest versions of packages from various package managers and registries.

```bash
# Run the package version MCP server
megatool run package-version
```

Supported package managers and registries:
- NPM (Node.js)
- PyPI (Python)
- Maven and Gradle (Java)
- Go Modules
- Swift Packages
- Docker Images
- AWS Bedrock Models

See the [Package Version README](cmd/megatool-package-version/README.md) for more details.

## Configuration

Each MCP server can be configured using the `--configure` flag:

```bash
megatool run <server> --configure
```

This will prompt for the necessary configuration values and store them securely.

## Development

### Project Structure

```
megatool/
├── cmd/
│   ├── megatool/
│   │   └── main.go                  # Main entry point, dispatches to server binaries
│   ├── megatool-calculator/
│   │   └── main.go                  # Calculator MCP server implementation
│   ├── megatool-github/
│   │   └── main.go                  # GitHub MCP server implementation
│   └── megatool-package-version/
│       ├── main.go                  # Package Version MCP server implementation
│       ├── README.md                # Package Version MCP server documentation
│       └── handlers/                # Package Version handlers
│           ├── types.go             # Common types
│           ├── utils.go             # Utility functions
│           ├── npm.go               # NPM handler
│           ├── python.go            # Python handler
│           ├── java.go              # Java handler
│           ├── go.go                # Go handler
│           ├── bedrock.go           # AWS Bedrock handler
│           ├── docker.go            # Docker handler
│           └── swift.go             # Swift handler
├── internal/
│   ├── config/
│   │   └── config.go                # Configuration management
│   └── utils/
│       └── utils.go                 # Shared utility functions
├── go.mod                           # Go module definition
├── go.sum                           # Go module checksums
├── justfile                         # Just task runner definitions
└── .mise.toml                       # Mise configuration for Go toolchain
```

### Development Tasks

The project includes a `justfile` with common development tasks:

```bash
# Initialize the project (install dependencies)
just init

# Build all binaries
just build

# Run tests
just test

# Run code quality checks
just lint

# Install binaries
just install

# Run the calculator MCP server (for development)
just run-calculator

# Run the GitHub MCP server (for development)
just run-github

# Run the Package Version MCP server (for development)
just run-package-version

# Configure the GitHub MCP server
just configure-github

# Clean build artifacts
just clean
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.
