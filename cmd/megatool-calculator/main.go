package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/pflag"
)

func main() {
	// Parse command line flags
	var showHelp bool
	pflag.BoolVarP(&showHelp, "help", "h", false, "Show help")
	pflag.Parse()

	// Show help if requested
	if showHelp {
		fmt.Println("MegaTool Calculator MCP Server")
		fmt.Println()
		fmt.Println("Usage: megatool calculator [flags]")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  --help, -h    Show this help message")
		os.Exit(0)
	}

	// Create a new MCP server
	s := server.NewMCPServer(
		"MegaTool Calculator",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	// Add calculator tool
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
	s.AddTool(calculatorTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		op, ok := request.Params.Arguments["operation"].(string)
		if !ok {
			return mcp.NewToolResultError("operation must be a string"), nil
		}

		x, ok := request.Params.Arguments["x"].(float64)
		if !ok {
			return mcp.NewToolResultError("x must be a number"), nil
		}

		y, ok := request.Params.Arguments["y"].(float64)
		if !ok {
			return mcp.NewToolResultError("y must be a number"), nil
		}

		// Perform calculation
		var result float64
		switch op {
		case "add":
			result = x + y
		case "subtract":
			result = x - y
		case "multiply":
			result = x * y
		case "divide":
			if y == 0 {
				return mcp.NewToolResultError("Division by zero is not allowed"), nil
			}
			result = x / y
		default:
			return mcp.NewToolResultError(fmt.Sprintf("Unknown operation: %s", op)), nil
		}

		// Return result
		return mcp.NewToolResultText(fmt.Sprintf("Result: %.2f", result)), nil
	})

	// Start the server
	fmt.Fprintln(os.Stderr, "Starting MegaTool Calculator MCP Server...")
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
