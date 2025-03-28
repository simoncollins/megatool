package mcpserver

import (
	"github.com/mark3labs/mcp-go/server"
)

// MCPServerHandler defines the interface for MCP server implementations
type MCPServerHandler interface {
	// Initialize sets up the server and registers all tools/resources
	Initialize(s *server.MCPServer) error

	// Name returns the display name of the server
	Name() string

	// Capabilities returns server capability options
	Capabilities() []server.ServerOption
}
