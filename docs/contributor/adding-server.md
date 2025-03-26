# Adding a New MCP Server

This guide provides step-by-step instructions for adding a new MCP server to MegaTool.

## Overview

Adding a new MCP server to MegaTool involves:

1. Creating a new directory for the server
2. Implementing the MCP server
3. Updating the main dispatcher to recognize the new server
4. Adding documentation

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

## Step 2: Implement the MCP Server

Create a `main.go` file in your server directory:

```go
// cmd/megatool-<server-name>/main.go

package main

import (
	"fmt"
	"os"

	"github.com/modelcontextprotocol/sdk/server"
	"github.com/modelcontextprotocol/sdk/server/stdio"
	"github.com/modelcontextprotocol/sdk/types"
	"github.com/simoncollins/megatool/internal/config"
)

func main() {
	// Parse command-line flags
	configureMode := false
	for _, arg := range os.Args[1:] {
		if arg == "--configure" {
			configureMode = true
			break
		}
	}

	// Handle configuration mode
	if configureMode {
		if err := configure(); err != nil {
			fmt.Fprintf(os.Stderr, "Error configuring server: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Create and run the server
	if err := runServer(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running server: %v\n", err)
		os.Exit(1)
	}
}

func configure() error {
	// Implement configuration logic here
	// Use the config package to store configuration securely
	return nil
}

func runServer() error {
	// Create a new MCP server
	s := server.New(
		server.ServerInfo{
			Name:    "megatool-<server-name>",
			Version: "0.1.0",
		},
		server.ServerOptions{
			Capabilities: server.Capabilities{
				Tools:     true,
				Resources: true,
			},
		},
	)

	// Set up request handlers
	setupHandlers(s)

	// Set up error handler
	s.OnError = func(err error) {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
	}

	// Connect to stdio transport
	transport := stdio.New()
	return s.Connect(transport)
}

func setupHandlers(s *server.Server) {
	// List Tools handler
	s.SetRequestHandler(types.ListToolsRequestSchema, func(req types.Request) (types.Response, error) {
		return types.ListToolsResponse{
			Tools: []types.Tool{
				{
					Name:        "example_tool",
					Description: "An example tool",
					InputSchema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"param1": map[string]interface{}{
								"type":        "string",
								"description": "A string parameter",
							},
						},
						"required": []string{"param1"},
					},
				},
			},
		}, nil
	})

	// Call Tool handler
	s.SetRequestHandler(types.CallToolRequestSchema, func(req types.Request) (types.Response, error) {
		params := req.Params.(types.CallToolParams)
		
		// Handle different tools
		switch params.Name {
		case "example_tool":
			// Implement tool logic here
			return types.CallToolResponse{
				Content: []types.Content{
					{
						Type: "text",
						Text: "Example tool response",
					},
				},
			}, nil
		default:
			return nil, types.NewError(types.ErrorCodeMethodNotFound, "Unknown tool: "+params.Name)
		}
	})

	// List Resources handler (optional)
	s.SetRequestHandler(types.ListResourcesRequestSchema, func(req types.Request) (types.Response, error) {
		return types.ListResourcesResponse{
			Resources: []types.Resource{
				{
					URI:         "example://resource",
					Name:        "Example Resource",
					Description: "An example resource",
					MimeType:    "text/plain",
				},
			},
		}, nil
	})

	// Read Resource handler (optional)
	s.SetRequestHandler(types.ReadResourceRequestSchema, func(req types.Request) (types.Response, error) {
		params := req.Params.(types.ReadResourceParams)
		
		// Handle different resources
		switch params.URI {
		case "example://resource":
			return types.ReadResourceResponse{
				Contents: []types.ResourceContent{
					{
						URI:      params.URI,
						MimeType: "text/plain",
						Text:     "Example resource content",
					},
				},
			}, nil
		default:
			return nil, types.NewError(types.ErrorCodeInvalidRequest, "Unknown resource: "+params.URI)
		}
	})
}
```

Customize this template for your specific server:

1. Replace `<server-name>` with your server's name
2. Implement the `configure()` function if your server needs configuration
3. Define the tools and resources your server provides
4. Implement the logic for each tool and resource

## Step 3: Update the Main Dispatcher

Update the main dispatcher to recognize your new server. This typically involves:

1. Adding your server to the list of available servers in `cmd/megatool/commands.go`
2. Ensuring the dispatcher can execute your server binary

## Step 4: Add Documentation

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

## Step 5: Build and Test

Build your new server:

```bash
just build
```

Test your server:

```bash
just run-<server-name>
```

## Step 6: Add Tests

Add tests for your server in a `_test.go` file:

```go
// cmd/megatool-<server-name>/main_test.go

package main

import (
	"testing"
	// Import other necessary packages
)

func TestExampleTool(t *testing.T) {
	// Implement tests for your tools
}

func TestExampleResource(t *testing.T) {
	// Implement tests for your resources
}
```

## Best Practices

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

Here's a simplified example of a Weather MCP server:

```go
// cmd/megatool-weather/main.go

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/sdk/server"
	"github.com/modelcontextprotocol/sdk/server/stdio"
	"github.com/modelcontextprotocol/sdk/types"
	"github.com/simoncollins/megatool/internal/config"
)

const (
	configKey = "weather_api_key"
)

func main() {
	// Parse command-line flags
	configureMode := false
	for _, arg := range os.Args[1:] {
		if arg == "--configure" {
			configureMode = true
			break
		}
	}

	// Handle configuration mode
	if configureMode {
		if err := configure(); err != nil {
			fmt.Fprintf(os.Stderr, "Error configuring weather server: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Create and run the server
	if err := runServer(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running weather server: %v\n", err)
		os.Exit(1)
	}
}

func configure() error {
	fmt.Print("Enter your weather API key: ")
	var apiKey string
	if _, err := fmt.Scanln(&apiKey); err != nil {
		return err
	}

	// Store the API key securely
	cfg := config.New("weather")
	return cfg.Set(configKey, apiKey)
}

func runServer() error {
	// Get the API key
	cfg := config.New("weather")
	apiKey, err := cfg.Get(configKey)
	if err != nil {
		return fmt.Errorf("API key not found. Please run with --configure first: %w", err)
	}

	// Create a new MCP server
	s := server.New(
		server.ServerInfo{
			Name:    "megatool-weather",
			Version: "0.1.0",
		},
		server.ServerOptions{
			Capabilities: server.Capabilities{
				Tools:     true,
				Resources: false,
			},
		},
	)

	// Set up request handlers
	setupHandlers(s, apiKey)

	// Set up error handler
	s.OnError = func(err error) {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
	}

	// Connect to stdio transport
	transport := stdio.New()
	return s.Connect(transport)
}

func setupHandlers(s *server.Server, apiKey string) {
	// List Tools handler
	s.SetRequestHandler(types.ListToolsRequestSchema, func(req types.Request) (types.Response, error) {
		return types.ListToolsResponse{
			Tools: []types.Tool{
				{
					Name:        "get_weather",
					Description: "Get the current weather for a location",
					InputSchema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{
								"type":        "string",
								"description": "The location to get weather for (city name)",
							},
							"units": map[string]interface{}{
								"type":        "string",
								"description": "The units to use (metric or imperial)",
								"enum":        []string{"metric", "imperial"},
								"default":     "metric",
							},
						},
						"required": []string{"location"},
					},
				},
			},
		}, nil
	})

	// Call Tool handler
	s.SetRequestHandler(types.CallToolRequestSchema, func(req types.Request) (types.Response, error) {
		params := req.Params.(types.CallToolParams)
		
		switch params.Name {
		case "get_weather":
			// Parse arguments
			args, ok := params.Arguments.(map[string]interface{})
			if !ok {
				return nil, types.NewError(types.ErrorCodeInvalidParams, "Invalid arguments")
			}
			
			location, ok := args["location"].(string)
			if !ok {
				return nil, types.NewError(types.ErrorCodeInvalidParams, "Location must be a string")
			}
			
			units := "metric"
			if u, ok := args["units"].(string); ok {
				units = u
			}
			
			// Call weather API
			weather, err := getWeather(apiKey, location, units)
			if err != nil {
				return nil, types.NewError(types.ErrorCodeInternalError, "Error getting weather: "+err.Error())
			}
			
			return types.CallToolResponse{
				Content: []types.Content{
					{
						Type: "text",
						Text: weather,
					},
				},
			}, nil
		default:
			return nil, types.NewError(types.ErrorCodeMethodNotFound, "Unknown tool: "+params.Name)
		}
	})
}

func getWeather(apiKey, location, units string) (string, error) {
	// Make API request to weather service
	url := fmt.Sprintf("https://api.example.com/weather?q=%s&units=%s&appid=%s", location, units, apiKey)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	// Parse response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	
	// Format weather information
	// This is a simplified example
	return fmt.Sprintf("Weather in %s: %v°C, %v", 
		location, 
		result["main"].(map[string]interface{})["temp"], 
		result["weather"].([]interface{})[0].(map[string]interface{})["description"]), nil
}
```

This example demonstrates:

1. Configuration handling for an API key
2. Defining a tool for getting weather information
3. Implementing the tool logic to call a weather API
4. Error handling and response formatting

## Conclusion

Adding a new MCP server to MegaTool is a straightforward process that involves creating a new server binary, implementing the MCP protocol, and updating the main dispatcher. By following this guide, you can extend MegaTool with new functionality that can be used by MCP clients.
