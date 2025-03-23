# Development Guide

This guide provides information for contributors who want to develop and extend MegaTool.

## Development Environment Setup

### Prerequisites

- Go 1.24 or later
- [Mise](https://github.com/jdx/mise) for managing the Go toolchain
- [Just](https://github.com/casey/just) for running tasks
- Git

### Setting Up Your Environment

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/megatool.git
   cd megatool
   ```

2. Initialize the development environment:
   ```bash
   just init
   ```

   This will set up the Go toolchain using Mise and install any required dependencies.

## Development Workflow

### Building

Build all binaries:

```bash
just build
```

This will build the main `megatool` binary and all server binaries in the `bin/` directory.

### Testing

Run all tests:

```bash
just test
```

Run tests with coverage:

```bash
just test-coverage
```

### Linting and Code Quality

Run linters and code quality checks:

```bash
just lint
```

This will run:
- `go fmt` to format the code
- `go vet` to check for potential issues
- `staticcheck` for additional static analysis

### Running Servers for Development

Run a specific server for development:

```bash
# Run the calculator server
just run-calculator

# Run the GitHub server
just run-github

# Run the package version server
just run-package-version
```

### Installing

Install the binaries to your Go bin directory:

```bash
just install
```

## Project Structure

See [Project Structure](project-structure.md) for a detailed overview of the codebase organization.

## Coding Standards

### Go Code Style

- Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` or `go fmt` to format your code
- Follow the standard Go naming conventions
- Write meaningful comments and documentation

### Commit Messages

Follow the conventional commit format:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

Where `<type>` is one of:
- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation changes
- `style`: Changes that do not affect the meaning of the code
- `refactor`: Code changes that neither fix a bug nor add a feature
- `perf`: Performance improvements
- `test`: Adding or modifying tests
- `chore`: Changes to the build process or auxiliary tools

### Error Handling

- Always handle errors properly; do not ignore errors
- Use descriptive error messages
- Consider using error wrapping for context

### Testing

- Write unit tests for critical components
- Aim for at least 80% test coverage
- Use table-driven tests where appropriate
- Mock external dependencies for testing

## Adding a New MCP Server

See [Adding a New Server](adding-server.md) for a detailed guide on adding a new MCP server to MegaTool.

## Security Considerations

- Store sensitive data using go-keyring
- Validate all user input
- Follow the principle of least privilege
- Be cautious with external dependencies

## Documentation

- Update documentation when code changes affect user-facing functionality
- Document public APIs with clear descriptions and examples
- Keep the README and other documentation up to date

## Release Process

1. Update version numbers in relevant files
2. Update the CHANGELOG.md file
3. Create a new Git tag for the release
4. Build release binaries
5. Create a GitHub release with the binaries

## Continuous Integration

The project uses GitHub Actions for continuous integration:

- Automated testing on multiple platforms
- Code quality checks
- Build verification

## Getting Help

If you need help with development:

- Check the existing documentation
- Look at the code for similar features
- Open an issue on GitHub for questions or problems
