# MegaTool

MegaTool is a command-line tool that implements multiple Model Context Protocol (MCP) servers, providing various utilities through a unified interface.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/simoncollins/megatool/actions/workflows/ci.yml/badge.svg)](https://github.com/simoncollins/megatool/actions/workflows/ci.yml)
[![Release](https://github.com/simoncollins/megatool/actions/workflows/release.yml/badge.svg)](https://github.com/simoncollins/megatool/actions/workflows/release.yml)

## Overview

MegaTool provides access to multiple MCP servers through a simple command-line interface. Each server offers specific functionality:

- **Calculator**: Perform arithmetic operations
- **GitHub**: Access GitHub repository and user information
- **Package Version**: Check latest versions of packages from various package managers

## Quick Installation

```bash
# From GitHub Releases (recommended)
# Download from https://github.com/simoncollins/megatool/releases
# Example for Linux:
tar -xzf megatool-v1.0.0-linux-amd64.tar.gz -C /usr/local/bin

# Using Go
go install github.com/simoncollins/megatool@latest

# From source
git clone https://github.com/simoncollins/megatool.git
cd megatool
just install
```

See the [detailed installation guide](docs/user/installation.md) for prerequisites and alternative methods.

## Basic Usage

```bash
# Run an MCP server in stdio mode (default)
megatool run <server-name>

# Run an MCP server in SSE mode
megatool run <server-name> --sse --port 8080

# Configure an MCP server
megatool run <server-name> --configure
```

Available servers:

- `calculator` - Simple calculator operations
- `github` - GitHub repository and user information
- `package-version` - Package version checker for multiple languages

## Examples

```bash
# Run the calculator server
megatool run calculator

# Configure the GitHub server (required before first use)
megatool run github --configure

# Run the package version server
megatool run package-version

# Run the calculator server in SSE mode on port 3000
megatool run calculator --sse --port 3000
```

## Documentation

### User Documentation

- [Installation Guide](docs/user/installation.md)
- [General Usage](docs/user/usage.md)
- [Calculator Server](docs/user/calculator.md)
- [GitHub Server](docs/user/github.md)
- [Package Version Server](docs/user/package-version.md)

### Contributor Documentation

- [Architecture Overview](docs/contributor/architecture.md)
- [Development Guide](docs/contributor/development.md)
- [Project Structure](docs/contributor/project-structure.md)
- [Adding a New Server](docs/contributor/adding-server.md)
- [Implementing SSE Mode](docs/contributor/sse-mode.md)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
