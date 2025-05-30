package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/megatool/internal/mcpserver"
	"github.com/urfave/cli/v2"
)

func main() {
	// Create a new GitHub server
	githubServer := NewGitHubServer()

	// Check if we should run in SSE mode
	sseMode := os.Getenv("MCP_SERVER_MODE") == "sse"
	port := os.Getenv("MCP_SERVER_PORT")
	baseURL := os.Getenv("MCP_SERVER_BASE_URL")

	// Define custom flags
	flags := []cli.Flag{
		&cli.BoolFlag{
			Name:    "configure",
			Aliases: []string{"c"},
			Usage:   "Configure GitHub MCP server",
		},
	}

	// Define custom action
	action := func(c *cli.Context) error {
		// Handle configuration mode
		if c.Bool("configure") {
			return githubServer.Configure()
		}

		// Load configuration
		if err := githubServer.LoadConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Run 'megatool github --configure' to configure the GitHub MCP server\n")
			return err
		}

		// Run the server in SSE mode if requested
		if sseMode {
			return runSSEServer(githubServer, port, baseURL)
		}

		// Run the server in stdio mode
		return mcpserver.CreateAndRunServer(githubServer)
	}

	// Create and run the CLI app
	app := mcpserver.NewCliApp(githubServer, flags, action)
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
		logger, _ := mcpserver.SetupLogger("github", os.Getpid())
		if logger != nil {
			logger.WithField("port", port).Info("SSE server listening")
		}
	}
	if err := sseServer.Start(":" + port); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
