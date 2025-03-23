# MegaTool Usage Guide

This guide covers the general usage of MegaTool and its command-line interface.

## Command Structure

MegaTool uses a simple command structure:

```
megatool [global options] command [command options] [arguments...]
```

The primary command for accessing MCP servers is `run`:

```
megatool run <server-name> [options]
```

## Global Options

These options apply to all MegaTool commands:

| Option | Description |
|--------|-------------|
| `--help`, `-h` | Show help information |
| `--version`, `-v` | Show version information |
| `--debug` | Enable debug logging |

## The `run` Command

The `run` command is used to start an MCP server:

```bash
megatool run <server-name> [options]
```

Where `<server-name>` is one of the available MCP servers:
- `calculator`
- `github`
- `package-version`

### Options for the `run` Command

| Option | Description |
|--------|-------------|
| `--configure` | Configure the server before running |
| `--help`, `-h` | Show help information for the server |

## Configuration

Some MCP servers require configuration before they can be used. You can configure a server using the `--configure` flag:

```bash
megatool run <server-name> --configure
```

This will prompt you for the necessary configuration values and store them securely using your system's keyring.

## Server-Specific Usage

Each MCP server has its own specific usage and capabilities:

- [Calculator Server](calculator.md)
- [GitHub Server](github.md)
- [Package Version Server](package-version.md)

## Examples

### Running the Calculator Server

```bash
megatool run calculator
```

The calculator server provides basic arithmetic operations and doesn't require configuration.

### Configuring and Running the GitHub Server

```bash
# First-time setup
megatool run github --configure

# After configuration
megatool run github
```

The GitHub server requires a Personal Access Token (PAT) for authentication with the GitHub API.

### Running the Package Version Server

```bash
megatool run package-version
```

The Package Version server checks for the latest versions of packages from various package managers and registries.

## Using MegaTool with MCP Clients

MegaTool is designed to be used with MCP clients, such as Claude or other AI assistants that support the Model Context Protocol. When running an MCP server, it communicates with the client over stdio.

To use MegaTool with an MCP client:

1. Start the MCP server:
   ```bash
   megatool run <server-name>
   ```

2. The server will wait for MCP requests from the client.

3. The client can then use the server's tools and resources through the MCP interface.

## Troubleshooting

### Common Issues

#### Server Not Starting

If a server fails to start, check:

1. That you have the necessary permissions
2. That the server is properly configured (if required)
3. That there are no conflicting processes using the same resources

#### Configuration Issues

If you encounter issues with configuration:

1. Try reconfiguring the server:
   ```bash
   megatool run <server-name> --configure
   ```

2. Check that you have the necessary credentials or API keys

#### Communication Issues

If the MCP client cannot communicate with the server:

1. Ensure that the server is running
2. Check that the client is properly configured to use the server
3. Look for any error messages in the server output
