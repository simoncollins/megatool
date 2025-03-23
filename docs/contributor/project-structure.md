# Project Structure

This document provides a detailed overview of the MegaTool project structure and codebase organization.

## Directory Structure

```
megatool/
├── .clinerules                    # Project-specific rules for Cline
├── .gitignore                     # Git ignore file
├── .mise.toml                     # Mise configuration for Go toolchain
├── go.mod                         # Go module definition
├── go.sum                         # Go module checksums
├── justfile                       # Just task runner definitions
├── LICENSE                        # Project license (MIT)
├── README.md                      # Project README
├── bin/                           # Build output directory
├── cmd/                           # Command-line applications
│   ├── megatool/                  # Main dispatcher binary
│   │   ├── commands.go            # Command definitions
│   │   ├── display.go             # Display and output formatting
│   │   ├── execute.go             # Command execution
│   │   └── main.go                # Main entry point
│   ├── megatool-calculator/       # Calculator MCP server
│   │   └── main.go                # Calculator server implementation
│   ├── megatool-github/           # GitHub MCP server
│   │   └── main.go                # GitHub server implementation
│   └── megatool-package-version/  # Package version MCP server
│       ├── main.go                # Package version server entry point
│       ├── README.md              # Package version server documentation
│       └── handlers/              # Package version handlers
│           ├── types.go           # Common types
│           ├── utils.go           # Utility functions
│           ├── npm.go             # NPM handler
│           ├── python.go          # Python handler
│           ├── java.go            # Java handler
│           ├── go.go              # Go handler
│           ├── bedrock.go         # AWS Bedrock handler
│           ├── docker.go          # Docker handler
│           └── swift.go           # Swift handler
└── internal/                      # Internal packages (not exported)
    ├── config/                    # Configuration management
    │   ├── config.go              # Configuration implementation
    │   └── config_test.go         # Configuration tests
    └── utils/                     # Shared utility functions
        ├── process.go             # Process management utilities
        ├── storage.go             # Storage utilities
        ├── utils.go               # General utilities
        └── utils_test.go          # Utility tests
```

## Key Components

### Main Dispatcher (`cmd/megatool/`)

The main dispatcher is responsible for parsing command-line arguments and executing the appropriate MCP server binary.

- **main.go**: Entry point for the application
- **commands.go**: Defines the available commands and their options
- **display.go**: Handles output formatting and display
- **execute.go**: Manages the execution of MCP server binaries

### Calculator Server (`cmd/megatool-calculator/`)

A simple MCP server that provides basic arithmetic operations.

- **main.go**: Implements the calculator MCP server

### GitHub Server (`cmd/megatool-github/`)

An MCP server that provides access to GitHub repository and user information.

- **main.go**: Implements the GitHub MCP server

### Package Version Server (`cmd/megatool-package-version/`)

An MCP server that checks for the latest versions of packages from various package managers and registries.

- **main.go**: Entry point for the package version server
- **handlers/**: Package-specific handlers for different package managers
  - **types.go**: Common type definitions
  - **utils.go**: Shared utility functions
  - **npm.go**: Handler for NPM packages
  - **python.go**: Handler for Python packages
  - **java.go**: Handler for Java packages (Maven and Gradle)
  - **go.go**: Handler for Go packages
  - **bedrock.go**: Handler for AWS Bedrock models
  - **docker.go**: Handler for Docker images
  - **swift.go**: Handler for Swift packages

### Configuration Management (`internal/config/`)

Handles configuration for MCP servers, including secure storage of sensitive data.

- **config.go**: Configuration implementation
- **config_test.go**: Tests for the configuration package

### Utility Functions (`internal/utils/`)

Shared utility functions used across the project.

- **process.go**: Utilities for process management
- **storage.go**: Utilities for storage operations
- **utils.go**: General utility functions
- **utils_test.go**: Tests for utility functions

## Code Organization Principles

### 1. Command Pattern

The main `megatool` binary uses the command pattern to organize its functionality:

- Each command is defined in `commands.go`
- Commands can have subcommands and flags
- The `run` command is the primary command for executing MCP servers

### 2. Separation of Concerns

Each MCP server is implemented as a separate binary with a clear focus:

- The calculator server focuses on arithmetic operations
- The GitHub server focuses on GitHub API integration
- The package version server focuses on package version checking

### 3. Internal Packages

Packages in the `internal/` directory are not exported and are only used within the project:

- `config`: Configuration management
- `utils`: Shared utility functions

### 4. Handler Pattern

The package version server uses a handler pattern to organize its functionality:

- Each package manager has its own handler
- Handlers implement a common interface
- The main server routes requests to the appropriate handler

## Build System

The project uses Just as a task runner:

- `just build`: Build all binaries
- `just test`: Run tests
- `just lint`: Run linters
- `just install`: Install binaries
- `just run-<server>`: Run a specific server for development

## Testing Strategy

The project uses Go's built-in testing framework:

- Unit tests for individual functions and components
- Integration tests for end-to-end functionality
- Table-driven tests for testing multiple inputs and outputs

## Dependency Management

The project uses Go modules for dependency management:

- `go.mod`: Defines the module and its dependencies
- `go.sum`: Contains the expected cryptographic checksums of the content of specific module versions

## Toolchain Management

The project uses Mise for managing the Go toolchain:

- `.mise.toml`: Defines the required Go version and other tools
