# Implementing SSE Mode in MCP Servers

This guide explains how to implement Server-Sent Events (SSE) mode in your MCP servers.

## Overview

The Model Context Protocol (MCP) supports two transport types:

1. **Standard Input/Output (stdio)** - The default transport that communicates through standard input and output streams.
2. **Server-Sent Events (SSE)** - An HTTP-based transport that enables server-to-client streaming with HTTP POST requests for client-to-server communication.

MegaTool now supports running MCP servers in SSE mode, which allows them to be accessed over HTTP.

## Using SSE Mode

To run an MCP server in SSE mode, use the `--sse` flag with the `run` command:

```bash
megatool run <server-name> --sse [--port <port>] [--base-url <base-url>]
```

Where:

- `<port>` is the port to use for the HTTP server (default: 8080)
- `<base-url>` is the base URL for the server (default: http://localhost:<port>)

## Implementing SSE Mode in Your Server

To support SSE mode in your MCP server, you need to:

1. Check for environment variables set by MegaTool
2. Create and run an SSE server if SSE mode is enabled

### Example Implementation

Here's an example of how to implement SSE mode in your server:

```go
package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/megatool/internal/mcpserver"
	"github.com/urfave/cli/v2"
)

func main() {
	// Create your server handler
	myServer := NewMyServer()

	// Check if we should run in SSE mode
	sseMode := os.Getenv("MCP_SERVER_MODE") == "sse"
	port := os.Getenv("MCP_SERVER_PORT")
	baseURL := os.Getenv("MCP_SERVER_BASE_URL")

	// Create and run the CLI app with custom action if in SSE mode
	var action func(c *cli.Context) error
	if sseMode {
		action = func(c *cli.Context) error {
			return runSSEServer(myServer, port, baseURL)
		}
	}

	app := mcpserver.NewCliApp(myServer, nil, action)
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runSSEServer runs the server in SSE mode
func runSSEServer(handler mcpserver.MCPServerHandler, port, baseURL string) error {
	// Create a new MCP server
	s := server.NewMCPServer(
		handler.Name(),
		mcpserver.Version,
		handler.Capabilities()...,
	)

	// Initialize the server with the handler
	if err := handler.Initialize(s); err != nil {
		return fmt.Errorf("failed to initialize server: %w", err)
	}

	// Create an SSE server with default endpoints
	sseServer := server.NewSSEServer(s,
		server.WithBaseURL(baseURL),
		server.WithSSEEndpoint("/sse"),
		server.WithMessageEndpoint("/message"))

	// Start the SSE server
	fmt.Printf("SSE server listening on :%s\n", port)
	if err := sseServer.Start(":" + port); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
```

### Environment Variables

MegaTool sets the following environment variables when running a server in SSE mode:

| Variable | Description |
|----------|-------------|
| `MCP_SERVER_MODE` | Set to `sse` when running in SSE mode |
| `MCP_SERVER_PORT` | The port to use for the HTTP server |
| `MCP_SERVER_BASE_URL` | The base URL for the server |

## Server Support

All built-in MCP servers in MegaTool now support SSE mode:

- `calculator` - Simple calculator operations
- `github` - GitHub repository and user information
- `package-version` - Package version checker for multiple languages
- `example` - Example server demonstrating SSE mode implementation

You can run any of these servers in SSE mode:

```bash
megatool run calculator --sse --port 3000
megatool run github --sse --port 8080
megatool run package-version --sse --port 8081
megatool run example --sse --port 8082
```

The `megatool-example` server specifically demonstrates how to implement SSE mode in a custom server. You can find the source code in the `cmd/megatool-example` directory as a reference implementation.

### SSE Endpoints

When running in SSE mode, the server exposes two main endpoints:

1. **SSE Endpoint**: `/sse` - This is where clients connect to receive server-sent events.
2. **Message Endpoint**: `/message` - This is where clients send messages to the server.

For example, if you run a server on port 8080, the endpoints would be:

- SSE Endpoint: `http://localhost:8080/sse`
- Message Endpoint: `http://localhost:8080/message`

Clients should connect to the SSE endpoint to receive events and send messages to the message endpoint.

## Best Practices

1. **Always check for SSE mode**: Check the `MCP_SERVER_MODE` environment variable to determine if the server should run in SSE mode.

2. **Use the provided port and base URL**: Use the port and base URL provided by MegaTool through environment variables.

3. **Handle both transport types**: Your server should be able to handle both stdio and SSE transport types.

4. **Log the server mode**: Log whether the server is running in stdio or SSE mode to help with debugging.

5. **Provide appropriate error messages**: If there are any issues starting the SSE server, provide clear error messages.

## Security Considerations

When running an MCP server in SSE mode, consider the following security aspects:

1. **Network exposure**: SSE mode exposes your server over HTTP, which means it could be accessible to other machines on the network.
2. **Authentication**: Consider implementing authentication for your SSE server if it will be exposed to untrusted networks.
3. **HTTPS**: For production use, consider using HTTPS instead of HTTP by setting up a reverse proxy with TLS termination.
4. **Rate limiting**: Implement rate limiting to prevent abuse of your server.

## Troubleshooting

### Common Issues

#### Port Already in Use

If you see an error like "address already in use", it means another process is already using the specified port. Try using a different port:

```bash
megatool run <server-name> --sse --port 3001
```

#### Network Access Issues

If clients cannot connect to your SSE server, check:

1. That the server is running and listening on the correct port
2. That there are no firewall rules blocking access to the port
3. That the base URL is correctly set and accessible to clients
