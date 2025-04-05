package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/megatool/internal/mcpserver"
	"github.com/urfave/cli/v2"
)

func main() {
	// Create a new example server
	exampleServer := NewExampleServer()

	// Check if we should run in SSE mode
	sseMode := os.Getenv("MCP_SERVER_MODE") == "sse"
	port := os.Getenv("MCP_SERVER_PORT")
	baseURL := os.Getenv("MCP_SERVER_BASE_URL")

	// Create and run the CLI app with custom action if in SSE mode
	var action func(c *cli.Context) error
	if sseMode {
		action = func(c *cli.Context) error {
			return runSSEServer(exampleServer, port, baseURL)
		}
	}

	app := mcpserver.NewCliApp(exampleServer, nil, action)
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
	// Check if we're in help mode
	helpMode := os.Getenv("MCP_HELP_MODE") == "true"

	// Only set up logging if not in help mode
	if !helpMode {
		logger, _ := mcpserver.SetupLogger("example", os.Getpid())
		if logger != nil {
			logger.WithField("port", port).Info("SSE server listening")
		}
	}
	if err := sseServer.Start(":" + port); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
