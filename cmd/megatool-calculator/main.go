package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/megatool/internal/logging"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// logger is the global logger instance
var logger *logrus.Logger

func main() {
	// Initialize logger
	pid := os.Getpid()
	log, err := logging.NewLogger("calculator", pid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		// Continue with a default logger
		logger = logrus.New()
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	} else {
		logger = log.Logger
	}

	logger.WithFields(logrus.Fields{
		"pid": pid,
	}).Info("Starting calculator MCP server")

	app := &cli.App{
		Name:  "megatool-calculator",
		Usage: "MegaTool Calculator MCP Server",
		Action: func(c *cli.Context) error {
			// Create a new MCP server
			s := server.NewMCPServer(
				"MegaTool Calculator",
				"1.0.0",
				server.WithToolCapabilities(true),
				// Don't use the built-in logging since we have our own
				// server.WithLogging(),
			)

			logger.Info("Initialized MCP server")

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
				// Log the request
				logger.WithFields(logrus.Fields{
					"tool":      "calculate",
					"arguments": request.Params.Arguments,
				}).Info("Received calculation request")

				// Extract parameters
				op, ok := request.Params.Arguments["operation"].(string)
				if !ok {
					logger.WithField("error", "operation must be a string").Error("Invalid parameter")
					return mcp.NewToolResultError("operation must be a string"), nil
				}

				x, ok := request.Params.Arguments["x"].(float64)
				if !ok {
					logger.WithField("error", "x must be a number").Error("Invalid parameter")
					return mcp.NewToolResultError("x must be a number"), nil
				}

				y, ok := request.Params.Arguments["y"].(float64)
				if !ok {
					logger.WithField("error", "y must be a number").Error("Invalid parameter")
					return mcp.NewToolResultError("y must be a number"), nil
				}

				// Perform calculation
				var result float64
				switch op {
				case "add":
					result = x + y
					logger.WithFields(logrus.Fields{
						"operation": "add",
						"x":         x,
						"y":         y,
						"result":    result,
					}).Info("Calculation performed")
				case "subtract":
					result = x - y
					logger.WithFields(logrus.Fields{
						"operation": "subtract",
						"x":         x,
						"y":         y,
						"result":    result,
					}).Info("Calculation performed")
				case "multiply":
					result = x * y
					logger.WithFields(logrus.Fields{
						"operation": "multiply",
						"x":         x,
						"y":         y,
						"result":    result,
					}).Info("Calculation performed")
				case "divide":
					if y == 0 {
						logger.WithFields(logrus.Fields{
							"operation": "divide",
							"x":         x,
							"y":         y,
							"error":     "division by zero",
						}).Error("Calculation error")
						return mcp.NewToolResultError("Division by zero is not allowed"), nil
					}
					result = x / y
					logger.WithFields(logrus.Fields{
						"operation": "divide",
						"x":         x,
						"y":         y,
						"result":    result,
					}).Info("Calculation performed")
				default:
					logger.WithFields(logrus.Fields{
						"operation": op,
						"error":     "unknown operation",
					}).Error("Calculation error")
					return mcp.NewToolResultError(fmt.Sprintf("Unknown operation: %s", op)), nil
				}

				// Return result
				return mcp.NewToolResultText(fmt.Sprintf("Result: %.2f", result)), nil
			})

			// Start the server
			logger.Info("Starting MCP server on stdio")
			if err := server.ServeStdio(s); err != nil {
				logger.WithError(err).Error("Server error")
				return fmt.Errorf("server error: %w", err)
			}

			logger.Info("MCP server stopped")
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.WithError(err).Error("Application error")
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
