# MegaTool

MegaTool is a command-line tool that implements multiple Model Context Protocol (MCP) servers. Each MCP server is accessible via subcommands (e.g., `megatool github`, `megatool calculator`). The tool uses separate binaries for each MCP server and a main binary to dispatch to the correct one based on the subcommand.

## Features

- **Multiple MCP Servers**: Each server runs as a separate binary (e.g., `megatool-github`, `megatool-calculator`).
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
megatool <server> [flags]
```

Where `<server>` is one of the available MCP servers:
- `calculator`: Simple calculator MCP server
- `github`: GitHub MCP server

### Calculator MCP Server

The Calculator MCP server provides basic arithmetic operations.

```bash
# Run the calculator MCP server
megatool calculator
```

### GitHub MCP Server

The GitHub MCP server provides information about GitHub repositories and users.

```bash
# Configure the GitHub MCP server (required before first use)
megatool github --configure

# Run the GitHub MCP server
megatool github
```

## Configuration

Each MCP server can be configured using the `--configure` flag:

```bash
megatool <server> --configure
```

This will prompt for the necessary configuration values and store them securely.

## Development

### Project Structure

```
megatool/
├── cmd/
│   ├── megatool/
│   │   └── main.go          # Main entry point, dispatches to server binaries
│   ├── megatool-calculator/
│   │   └── main.go          # Calculator MCP server implementation
│   └── megatool-github/
│       └── main.go          # GitHub MCP server implementation
├── internal/
│   ├── config/
│   │   └── config.go        # Configuration management
│   └── utils/
│       └── utils.go         # Shared utility functions
├── go.mod                   # Go module definition
├── go.sum                   # Go module checksums
├── justfile                 # Just task runner definitions
└── .mise.toml               # Mise configuration for Go toolchain
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

# Configure the GitHub MCP server
just configure-github

# Clean build artifacts
just clean
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.
