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

// CalculatorServer implements the MCPServerHandler interface for the calculator server
type CalculatorServer struct {
	logger *logrus.Logger
}

// NewCalculatorServer creates a new calculator server
func NewCalculatorServer() *CalculatorServer {
	return &CalculatorServer{}
}

// Name returns the display name of the server
func (s *CalculatorServer) Name() string {
	return "Calculator"
}

// Capabilities returns the server capabilities
func (s *CalculatorServer) Capabilities() []server.ServerOption {
	return []server.ServerOption{
		server.WithToolCapabilities(true),
	}
}

// Initialize sets up the server
func (s *CalculatorServer) Initialize(srv *server.MCPServer) error {
	// Set up the logger
	pid := os.Getpid()
	logger, err := mcpserver.SetupLogger("calculator", pid)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	s.logger = logger

	s.logger.WithFields(logrus.Fields{
		"pid": pid,
	}).Info("Starting calculator MCP server")

	// Register calculator tool
	s.registerCalculatorTool(srv)

	s.logger.Info("Calculator server initialized")
	return nil
}

// registerCalculatorTool registers the calculator tool
func (s *CalculatorServer) registerCalculatorTool(srv *server.MCPServer) {
	// Create calculator tool
	calculatorTool := mcp.NewTool("calculate",
		mcp.WithDescription("Perform basic arithmetic calculations"),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("The arithmetic operation to perform"),
			mcp.Enum("add", "subtract", "multiply", "divide"),
		),
		mcp.WithNumber("x",
			mcp.Required(),
			mcp.Description("First number"),
		),
		mcp.WithNumber("y",
			mcp.Required(),
			mcp.Description("Second number"),
		),
	)

	// Add calculator tool handler
	srv.AddTool(calculatorTool, s.handleCalculate)
}

// handleCalculate handles the calculate tool
func (s *CalculatorServer) handleCalculate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcpserver.HandleToolRequest(ctx, request, s.performCalculation, s.logger)
}

// performCalculation performs the calculation
func (s *CalculatorServer) performCalculation(ctx context.Context, args map[string]interface{}) (string, error) {
	// Extract parameters
	op, ok := mcpserver.ExtractStringParam(args, "operation", s.logger)
	if !ok {
		return "", fmt.Errorf("operation must be a string")
	}

	x, ok := mcpserver.ExtractNumberParam(args, "x", s.logger)
	if !ok {
		return "", fmt.Errorf("x must be a number")
	}

	y, ok := mcpserver.ExtractNumberParam(args, "y", s.logger)
	if !ok {
		return "", fmt.Errorf("y must be a number")
	}

	// Perform calculation
	var result float64
	switch op {
	case "add":
		result = x + y
		s.logger.WithFields(logrus.Fields{
			"operation": "add",
			"x":         x,
			"y":         y,
			"result":    result,
		}).Info("Calculation performed")
	case "subtract":
		result = x - y
		s.logger.WithFields(logrus.Fields{
			"operation": "subtract",
			"x":         x,
			"y":         y,
			"result":    result,
		}).Info("Calculation performed")
	case "multiply":
		result = x * y
		s.logger.WithFields(logrus.Fields{
			"operation": "multiply",
			"x":         x,
			"y":         y,
			"result":    result,
		}).Info("Calculation performed")
	case "divide":
		if y == 0 {
			s.logger.WithFields(logrus.Fields{
				"operation": "divide",
				"x":         x,
				"y":         y,
				"error":     "division by zero",
			}).Error("Calculation error")
			return "", fmt.Errorf("division by zero is not allowed")
		}
		result = x / y
		s.logger.WithFields(logrus.Fields{
			"operation": "divide",
			"x":         x,
			"y":         y,
			"result":    result,
		}).Info("Calculation performed")
	default:
		s.logger.WithFields(logrus.Fields{
			"operation": op,
			"error":     "unknown operation",
		}).Error("Calculation error")
		return "", fmt.Errorf("unknown operation: %s", op)
	}

	// Return result
	return fmt.Sprintf("Result: %.2f", result), nil
}
