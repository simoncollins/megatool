package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/megatool/internal/mcpserver"
	"github.com/sirupsen/logrus"
)

// ExampleServer implements the MCPServerHandler interface for the example server
type ExampleServer struct {
	logger *logrus.Logger
}

// NewExampleServer creates a new example server
func NewExampleServer() *ExampleServer {
	return &ExampleServer{}
}

// Name returns the display name of the server
func (s *ExampleServer) Name() string {
	return "Example"
}

// Capabilities returns the server capabilities
func (s *ExampleServer) Capabilities() []server.ServerOption {
	return []server.ServerOption{
		server.WithToolCapabilities(true),
	}
}

// Initialize sets up the server
func (s *ExampleServer) Initialize(srv *server.MCPServer) error {
	// Set up the logger
	pid := os.Getpid()
	logger, err := mcpserver.SetupLogger("example", pid)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	s.logger = logger

	s.logger.WithFields(logrus.Fields{
		"pid": pid,
	}).Info("Starting example MCP server")

	// Register example tool
	s.registerExampleTool(srv)

	s.logger.Info("Example server initialized")
	return nil
}

// registerExampleTool registers the example tool
func (s *ExampleServer) registerExampleTool(srv *server.MCPServer) {
	// Create example tool
	exampleTool := mcp.NewTool("echo",
		mcp.WithDescription("Echo a message back"),
		mcp.WithString("message",
			mcp.Required(),
			mcp.Description("The message to echo back"),
		),
	)

	// Add example tool handler
	srv.AddTool(exampleTool, s.handleEcho)
}

// handleEcho handles the echo tool
func (s *ExampleServer) handleEcho(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcpserver.HandleToolRequest(ctx, request, s.echoMessage, s.logger)
}

// echoMessage echoes the message back
func (s *ExampleServer) echoMessage(ctx context.Context, args map[string]interface{}) (string, error) {
	// Extract parameters
	message, ok := mcpserver.ExtractStringParam(args, "message", s.logger)
	if !ok {
		return "", fmt.Errorf("message must be a string")
	}

	s.logger.WithFields(logrus.Fields{
		"message": message,
	}).Info("Echo request received")

	// Return the message
	return fmt.Sprintf("Echo: %s", message), nil
}
