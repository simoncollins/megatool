# Adding a New MCP Server

This guide provides step-by-step instructions for adding a new MCP server to MegaTool using the `MCPServerHandler` interface.

## Overview

Adding a new MCP server to MegaTool involves:

1. Creating a new directory for the server
2. Implementing the `MCPServerHandler` interface
3. Creating a main function that uses the common utilities
4. Updating the main dispatcher to recognize the new server
5. Adding documentation

## Prerequisites

Before adding a new server, ensure you have:

- A clear understanding of the Model Context Protocol (MCP)
- Familiarity with Go programming
- A development environment set up (see [Development Guide](development.md))

## Step 1: Create a New Server Directory

Create a new directory for your server in the `cmd/` directory:

```bash
mkdir -p cmd/megatool-<server-name>
```

Replace `<server-name>` with the name of your server (e.g., `weather`, `translator`, etc.).

## Step 2: Implement the MCPServerHandler Interface

Create a server implementation file (e.g., `server.go` or `<server-name>.go`):

```go
// cmd/megatool-<server-name>/<server-name>.go

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

// MyServer implements the MCPServerHandler interface
type MyServer struct {
	logger *logrus.Logger
	// Add your server-specific fields here
}

// NewMyServer creates a new instance of your server
func NewMyServer() *MyServer {
	return &MyServer{}
}

// Name returns the display name of your server
func (s *MyServer) Name() string {
	return "MyServer" // Replace with your server's name
}

// Capabilities returns the server capabilities
func (s *MyServer) Capabilities() []server.ServerOption {
	return []server.ServerOption{
		server.WithToolCapabilities(true),
		// Add other capabilities as needed
		// server.WithResourceCapabilities(true, true),
	}
}

// Initialize sets up your server
func (s *MyServer) Initialize(srv *server.MCPServer) error {
	// Set up the logger
	pid := os.Getpid()
	logger, err := mcpserver.SetupLogger("myserver", pid)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	s.logger = logger

	s.logger.WithFields(logrus.Fields{
		"pid": pid,
	}).Info("Starting server")

	// Register your tools and resources
	s.registerTools(srv)
	
	s.logger.Info("Server initialized")
	return nil
}

// registerTools registers your tools with the MCP server
func (s *MyServer) registerTools(srv *server.MCPServer) {
	// Example tool
	myTool := mcp.NewTool("my_tool",
		mcp.WithDescription("Description of my tool"),
		mcp.WithString("param1",
			mcp.Required(),
			mcp.Description("Description of parameter 1"),
		),
	)

	// Add the tool handler
	srv.AddTool(myTool, s.handleMyTool)
}

// handleMyTool handles the my_tool tool
func (s *MyServer) handleMyTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Log the request
	mcpserver.LogToolRequest(s.logger, "my_tool", request.Params.Arguments)
	
	// Extract parameters
	param1, ok := mcpserver.ExtractStringParam(request.Params.Arguments, "param1", s.logger)
	if !ok {
		return mcpserver.NewErrorResult("param1 must be a string"), nil
	}
	
	// Process the request
	result := fmt.Sprintf("Result: %s", param1)
	
	// Return the result
	return mcp.NewToolResultText(result), nil
}
```

## Step 3: Create the Main Function

Create a `main.go` file in your server directory:

### Simple Server (No Configuration)

For servers that don't need configuration:

```go
// cmd/megatool-<server-name>/main.go

package main

import (
	"fmt"
	"os"

	"github.com/megatool/internal/mcpserver"
)

func main() {
	// Create a new server
	myServer := NewMyServer()

	// Create and run the CLI app
	app := mcpserver.NewCliApp(myServer, nil, nil)
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

### Server with Configuration

For servers that need configuration:

```go
// cmd/megatool-<server-name>/main.go

package main

import (
	"fmt"
	"os"

	"github.com/megatool/internal/mcpserver"
	"github.com/urfave/cli/v2"
)

func main() {
	// Create a new server
	myServer := NewMyServer()

	// Define custom flags
	flags := []cli.Flag{
		&cli.BoolFlag{
			Name:    "configure",
			Aliases: []string{"c"},
			Usage:   "Configure the server",
		},
	}

	// Define custom action
	action := func(c *cli.Context) error {
		// Handle configuration mode
		if c.Bool("configure") {
			return myServer.Configure()
		}

		// Load configuration
		if err := myServer.LoadConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Run 'megatool <server-name> --configure' to configure the server\n")
			return err
		}

		// Run the server
		return mcpserver.CreateAndRunServer(myServer)
	}

	// Create and run the CLI app
	app := mcpserver.NewCliApp(myServer, flags, action)
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

If your server needs configuration, add these methods to your server implementation:

```go
// Configure handles the configuration of the server
func (s *MyServer) Configure() error {
	fmt.Println("Configuring server...")
	// Implement configuration logic here
	// Use the config package to store configuration securely
	return nil
}

// LoadConfig loads the server configuration
func (s *MyServer) LoadConfig() error {
	// Load configuration
	// Use the config package to load configuration
	return nil
}
```

## Step 4: Update the Main Dispatcher

Update the main dispatcher to recognize your new server. This typically involves:

1. Adding your server to the list of available servers in `cmd/megatool/commands.go`
2. Ensuring the dispatcher can execute your server binary

## Step 5: Add Documentation

### User Documentation

Create a new file in `docs/user/` for your server:

```markdown
# <Server Name> MCP Server

Brief description of what your server does.

## Features

- Feature 1
- Feature 2
- ...

## Configuration

Instructions for configuring your server (if needed).

## Usage

Instructions for using your server.

## Available Tools

Description of the tools your server provides.

## Examples

Examples of how to use your server with an MCP client.
```

### Update README.md

Add your server to the list of available servers in the main README.md.

## Step 6: Build and Test

Build your new server:

```bash
just build
```

Test your server:

```bash
just run-<server-name>
```

## Step 7: Add Tests

Add tests for your server in a `_test.go` file:

```go
// cmd/megatool-<server-name>/<server-name>_test.go

package main

import (
	"testing"
	// Import other necessary packages
)

func TestMyTool(t *testing.T) {
	// Implement tests for your tools
}
```

## Best Practices

### Follow the MCPServerHandler Interface

Ensure your server correctly implements the `MCPServerHandler` interface:

- `Name()`: Return a descriptive name for your server
- `Capabilities()`: Return the appropriate server capabilities
- `Initialize()`: Set up your server, register tools and resources

### Use Common Utilities

Take advantage of the common utilities provided by the `internal/mcpserver` package:

- `mcpserver.SetupLogger()`: Create a standard logger
- `mcpserver.ExtractStringParam()`, `mcpserver.ExtractNumberParam()`, etc.: Extract and validate parameters
- `mcpserver.NewErrorResult()`: Create standardized error responses
- `mcpserver.LogToolRequest()`: Log tool requests in a consistent format

### Follow the MCP Specification

Ensure your server follows the Model Context Protocol specification:

- Tools should have clear names, descriptions, and input schemas
- Resources should have unique URIs and appropriate MIME types
- Error handling should follow the MCP error codes

### Security Considerations

- Store sensitive data using go-keyring
- Validate all user input
- Follow the principle of least privilege
- Be cautious with external dependencies

### Code Organization

- Keep your server code modular and maintainable
- Use appropriate error handling
- Follow Go best practices
- Document your code

### Configuration

If your server requires configuration:

- Use the `internal/config` package for consistent configuration handling
- Store sensitive data securely
- Provide clear configuration instructions

## Example: Weather MCP Server

Here's a simplified example of a Weather MCP server using the `MCPServerHandler` interface:

```go
// cmd/megatool-weather/weather.go

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/megatool/internal/config"
	"github.com/megatool/internal/mcpserver"
	"github.com/sirupsen/logrus"
)

// WeatherServer implements the MCPServerHandler interface
type WeatherServer struct {
	logger *logrus.Logger
	apiKey string
}

// NewWeatherServer creates a new weather server
func NewWeatherServer() *WeatherServer {
	return &WeatherServer{}
}

// Name returns the display name of the server
func (s *WeatherServer) Name() string {
	return "Weather"
}

// Capabilities returns the server capabilities
func (s *WeatherServer) Capabilities() []server.ServerOption {
	return []server.ServerOption{
		server.WithToolCapabilities(true),
	}
}

// Configure handles the configuration of the weather server
func (s *WeatherServer) Configure() error {
	fmt.Println("Configuring Weather MCP Server")
	fmt.Println()
	fmt.Print("Enter your weather API key: ")
	var apiKey string
	if _, err := fmt.Scanln(&apiKey); err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	// Store API key in keyring
	err := config.StoreSecure("weather", "api_key", apiKey)
	if err != nil {
		return fmt.Errorf("failed to store API key in keyring: %w", err)
	}

	fmt.Println("Weather configuration saved successfully")
	return nil
}

// LoadConfig loads the weather server configuration
func (s *WeatherServer) LoadConfig() error {
	// Get API key from keyring
	apiKey, err := config.GetSecure("weather", "api_key")
	if err != nil {
		return fmt.Errorf("failed to retrieve API key: %w", err)
	}

	s.apiKey = apiKey
	return nil
}

// Initialize sets up the server
func (s *WeatherServer) Initialize(srv *server.MCPServer) error {
	// Set up the logger
	pid := os.Getpid()
	logger, err := mcpserver.SetupLogger("weather", pid)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	s.logger = logger

	s.logger.WithFields(logrus.Fields{
		"pid": pid,
	}).Info("Starting weather MCP server")

	// Register weather tool
	s.registerWeatherTool(srv)

	s.logger.Info("Weather server initialized")
	return nil
}

// registerWeatherTool registers the weather tool
func (s *WeatherServer) registerWeatherTool(srv *server.MCPServer) {
	// Create weather tool
	weatherTool := mcp.NewTool("get_weather",
		mcp.WithDescription("Get the current weather for a location"),
		mcp.WithString("location",
			mcp.Required(),
			mcp.Description("The location to get weather for (city name)"),
		),
		mcp.WithString("units",
			mcp.Description("The units to use (metric or imperial)"),
			mcp.Enum("metric", "imperial"),
			mcp.DefaultString("metric"),
		),
	)

	// Add weather tool handler
	srv.AddTool(weatherTool, s.handleGetWeather)
}

// handleGetWeather handles the get_weather tool
func (s *WeatherServer) handleGetWeather(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Log the request
	mcpserver.LogToolRequest(s.logger, "get_weather", request.Params.Arguments)

	// Extract parameters
	location, ok := mcpserver.ExtractStringParam(request.Params.Arguments, "location", s.logger)
	if !ok {
		return mcpserver.NewErrorResult("location must be a string"), nil
	}

	units := "metric"
	if unitsParam, ok := request.Params.Arguments["units"].(string); ok {
		units = unitsParam
	}

	s.logger.WithFields(logrus.Fields{
		"location": location,
		"units":    units,
	}).Info("Getting weather")

	// Call weather API
	weather, err := s.getWeather(location, units)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"location": location,
			"units":    units,
			"error":    err.Error(),
		}).Error("Failed to get weather")
		return mcpserver.NewErrorResult(fmt.Sprintf("Failed to get weather: %v", err)), nil
	}

	s.logger.WithFields(logrus.Fields{
		"location": location,
		"units":    units,
	}).Info("Weather fetched successfully")

	// Return result
	return mcp.NewToolResultText(weather), nil
}

// getWeather gets the weather for a location
func (s *WeatherServer) getWeather(location, units string) (string, error) {
	// Make API request to weather service
	url := fmt.Sprintf("https://api.example.com/weather?q=%s&units=%s&appid=%s", location, units, s.apiKey)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to connect to weather API: %w", err)
	}
	defer resp.Body.Close()

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("weather API error (status %d)", resp.StatusCode)
	}

	// Parse response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse weather data: %w", err)
	}

	// Format weather information
	return fmt.Sprintf("Weather in %s: %.1f°C, %s",
		location,
		result["main"].(map[string]interface{})["temp"].(float64),
		result["weather"].([]interface{})[0].(map[string]interface{})["description"].(string)), nil
}
```

And the main function:

```go
// cmd/megatool-weather/main.go

package main

import (
	"fmt"
	"os"

	"github.com/megatool/internal/mcpserver"
	"github.com/urfave/cli/v2"
)

func main() {
	// Create a new weather server
	weatherServer := NewWeatherServer()

	// Define custom flags
	flags := []cli.Flag{
		&cli.BoolFlag{
			Name:    "configure",
			Aliases: []string{"c"},
			Usage:   "Configure Weather MCP server",
		},
	}

	// Define custom action
	action := func(c *cli.Context) error {
		// Handle configuration mode
		if c.Bool("configure") {
			return weatherServer.Configure()
		}

		// Load configuration
		if err := weatherServer.LoadConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Run 'megatool weather --configure' to configure the Weather MCP server\n")
			return err
		}

		// Run the server
		return mcpserver.CreateAndRunServer(weatherServer)
	}

	// Create and run the CLI app
	app := mcpserver.NewCliApp(weatherServer, flags, action)
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

## Conclusion

Adding a new MCP server to MegaTool using the `MCPServerHandler` interface is a straightforward process that involves implementing the interface, creating a main function, and updating the main dispatcher. By following this guide, you can extend MegaTool with new functionality that can be used by MCP clients.

The `MCPServerHandler` interface and common utilities provide a standardized approach to implementing MCP servers, reducing code duplication and ensuring consistency across all servers.
