# MegaTool Usage Guide

This guide covers the general usage of MegaTool and its command-line interface.

## Command Structure

MegaTool uses a simple command structure:

```
megatool [global options] command [command options] [arguments...]
```

The primary commands for working with MCP servers are:

- `run`: Start an MCP server
  ```
  megatool run <server-name> [options]
  ```

- `install`: Install an MCP server into a client's configuration
  ```
  megatool install --client <client-name> <server-name>
  ```

- `cleanup`: Clean up logs from MCP servers that are no longer running
  ```
  megatool cleanup [options]
  ```

## Global Options

These options apply to all MegaTool commands:

| Option | Description |
|--------|-------------|
| `--help`, `-h` | Show help information |
| `--version`, `-v` | Show version information of MegaTool |
| `--debug` | Enable debug logging |

### Version Information

You can check the current version of MegaTool using the `--version` or `-v` flag:

```bash
megatool --version
# Output: megatool version v1.0.0-alpha.4
```

This is useful for:
- Reporting issues (always include your version number)
- Ensuring compatibility with MCP servers
- Checking if you need to update to a newer release

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
| `--client` | Target MCP client (e.g., cline) |
| `--sse` | Run the server in SSE (Server-Sent Events) mode |
| `--port` | Port to use for SSE mode (default: 8080) |
| `--base-url` | Base URL for SSE mode (default: http://localhost:<port>) |
| `--help`, `-h` | Show help information for the server |

## The `install` Command

The `install` command is used to install an MCP server into a client's configuration:

```bash
megatool install --client <client-name> <server-name>
```

Where:
- `<client-name>` is the name of the MCP client (currently supports `cline`)
- `<server-name>` is one of the available MCP servers

### Options for the `install` Command

| Option | Description |
|--------|-------------|
| `--client`, `-c` | Target MCP client (e.g., cline) - required |
| `--help`, `-h` | Show help information |

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

MegaTool is designed to be used with MCP clients, such as Claude or other AI assistants that support the Model Context Protocol. MegaTool supports two transport modes: stdio and SSE (Server-Sent Events).

### Standard Input/Output (stdio) Mode

By default, MegaTool runs in stdio mode, which is suitable for direct integration with MCP clients:

1. Start the MCP server:
   ```bash
   megatool run <server-name>
   ```

2. The server will wait for MCP requests from the client over standard input/output.

3. The client can then use the server's tools and resources through the MCP interface.

### Server-Sent Events (SSE) Mode

SSE mode allows the MCP server to be accessed over HTTP, which is useful for web-based clients or when you need to expose the server over a network:

1. Start the MCP server in SSE mode:
   ```bash
   megatool run <server-name> --sse --port 8080
   ```

2. The server will start an HTTP server on the specified port (default: 8080).

3. Clients can connect to the server using the SSE transport at the specified URL (default: http://localhost:8080).

#### SSE Mode Options

| Option | Description |
|--------|-------------|
| `--sse` | Enable SSE mode |
| `--port` | Port to use for the HTTP server (default: 8080) |
| `--base-url` | Base URL for the server (default: http://localhost:<port>) |

#### Example: Running a Server in SSE Mode

```bash
# Run the calculator server in SSE mode on port 3000
megatool run calculator --sse --port 3000

# Run the GitHub server in SSE mode with a custom base URL
megatool run github --sse --base-url https://mcp.example.com
```

### Installing into a Client's Configuration

For a more integrated experience, you can install an MCP server into a client's configuration:

1. Install the server into the client's configuration:
   ```bash
   megatool install --client cline <server-name>
   ```

2. The server will be added to the client's configuration file.

3. The client will automatically start the server when needed.

#### Supported Clients

Currently, MegaTool supports the following MCP clients:

- `cline`: The VS Code Cline extension for Claude

## The `ps` Command

The `ps` command is used to list running MCP servers:

```bash
megatool ps [options]
```

### Options for the `ps` Command

| Option | Description |
|--------|-------------|
| `--format`, `-f` | Output format (table, json, csv) |
| `--fields` | Comma-separated list of fields to display (name, pid, uptime, client) |
| `--no-header` | Don't print header row |
| `--client` | Filter servers by client (e.g., cline) |

## The `stop` Command

The `stop` command is used to stop running MCP servers:

```bash
megatool stop <server-name> [options]
```

### Options for the `stop` Command

| Option | Description |
|--------|-------------|
| `--all` | Stop all instances of the specified server |
| `--pid` | Stop a specific instance by PID |
| `--client` | Filter servers by client (e.g., cline) |

## The `cleanup` Command

The `cleanup` command is used to clean up logs from MCP servers that are no longer running:

```bash
megatool cleanup [options]
```

This command will:
1. Remove log files for processes that are no longer running
2. Remove entire server log directories if all logs are older than the specified threshold
3. Clean up stale server records from the running-servers.json file

### Options for the `cleanup` Command

| Option | Description |
|--------|-------------|
| `--days`, `-d` | Remove logs older than this many days (default: 30) |
| `--dry-run` | Show what would be deleted without actually deleting |
| `--force`, `-f` | Skip confirmation prompts |
| `--verbose`, `-v` | Show detailed information about the cleanup process |

### Examples

```bash
# Show what would be cleaned up without actually deleting anything
megatool cleanup --dry-run

# Clean up logs older than 7 days with detailed output
megatool cleanup --days 7 --verbose

# Force cleanup without confirmation prompts
megatool cleanup --force
```

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
