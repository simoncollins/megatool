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
   git clone https://github.com/simoncollins/megatool.git
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

## Versioning and Releases

MegaTool follows [Semantic Versioning](https://semver.org/) (SemVer) for its release versioning. The version format is `vMAJOR.MINOR.PATCH[-PRERELEASE]`, where:

- `MAJOR` version increases for incompatible API changes
- `MINOR` version increases for backward-compatible functionality additions
- `PATCH` version increases for backward-compatible bug fixes
- `PRERELEASE` suffix (like `-alpha.4`) indicates pre-release versions

### Version Management

The current version is stored as a constant in `cmd/megatool/version.go` and is displayed when using the `--version` flag.

MegaTool provides Just commands to manage versioning:

#### Checking the Current Version

To see the current version in the codebase:

```bash
grep 'const Version =' cmd/megatool/version.go
```

To list all version tags:

```bash
just version-list
```

#### Creating a New Version

To update the version and create a matching Git tag:

```bash
just version 1.0.0
```

This command will:
1. Update the version constant in `cmd/megatool/version.go`
2. Commit the change with a conventional commit message (`chore: bump version to 1.0.0`)
3. Create an annotated Git tag (`v1.0.0`)

#### Publishing a Release

After creating a new version, you can publish the release with:

```bash
just release
```

This command will:
1. Push the code changes to the remote repository
2. Push the version tags to trigger the release workflow
3. Display a confirmation message with the released version

### Release Workflow

The complete release process is:

1. Ensure all changes for the release are committed
2. Update the version: `just version X.Y.Z`
3. Publish the release: `just release`
4. The CI/CD pipeline will automatically build and publish the release artifacts

### Pre-release Versions

For pre-release versions, use the appropriate suffix:

```bash
just version 1.1.0-alpha.1
```

### Version History

To view the history of all releases:

```bash
just version-list
```

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
