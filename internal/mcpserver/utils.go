package mcpserver

import (
	"fmt"

	"github.com/mark3labs/mcp-go/server"
	"github.com/megatool/internal/logging"
	"github.com/megatool/internal/version"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// Version is imported from the version package
var Version = version.Version

// SetupLogger creates and configures a logger for an MCP server
func SetupLogger(name string, pid int) (*logrus.Logger, error) {
	log, err := logging.NewLogger(name, pid)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	return log.Logger, nil
}

// CreateAndRunServer creates and runs an MCP server with the given handler
func CreateAndRunServer(handler MCPServerHandler) error {
	// Create a new MCP server
	s := server.NewMCPServer(
		handler.Name(),
		Version,
		handler.Capabilities()...,
	)

	// Initialize the server with the handler
	if err := handler.Initialize(s); err != nil {
		return fmt.Errorf("failed to initialize server: %w", err)
	}

	// Start the server
	if err := server.ServeStdio(s); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// NewCliApp creates a new CLI app for an MCP server
func NewCliApp(handler MCPServerHandler, flags []cli.Flag, action cli.ActionFunc) *cli.App {
	app := &cli.App{
		Name:    fmt.Sprintf("megatool-%s", handler.Name()),
		Usage:   fmt.Sprintf("MegaTool %s MCP Server", handler.Name()),
		Version: Version,
		Flags:   flags,
	}

	// If no custom action is provided, use the default action
	if action == nil {
		action = func(c *cli.Context) error {
			return CreateAndRunServer(handler)
		}
	}

	app.Action = action
	return app
}

// DefaultAction returns the default CLI action for an MCP server
func DefaultAction(handler MCPServerHandler) cli.ActionFunc {
	return func(c *cli.Context) error {
		return CreateAndRunServer(handler)
	}
}
