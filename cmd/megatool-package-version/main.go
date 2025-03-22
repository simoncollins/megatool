package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/megatool/cmd/megatool-package-version/handlers"
	"github.com/urfave/cli/v2"
)

const (
	// CacheTTL is the time-to-live for cached data (1 hour)
	CacheTTL = 1 * time.Hour
)

// Cache provides a simple in-memory cache with expiration
type Cache struct {
	data  map[string]interface{}
	times map[string]time.Time
	ttl   time.Duration
	mu    sync.RWMutex
}

// NewCache creates a new cache with the specified TTL
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		data:  make(map[string]interface{}),
		times: make(map[string]time.Time),
		ttl:   ttl,
	}
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	val, exists := c.data[key]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Since(c.times[key]) > c.ttl {
		return nil, false
	}

	return val, true
}

// Set stores a value in the cache
func (c *Cache) Set(key string, val interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = val
	c.times[key] = time.Now()
}

func main() {
	app := &cli.App{
		Name:  "megatool-package-version",
		Usage: "MegaTool Package Version MCP Server",
		Action: func(c *cli.Context) error {
			// Create a new MCP server
			s := server.NewMCPServer(
				"MegaTool Package Version",
				"1.0.0",
				server.WithToolCapabilities(true),
				server.WithLogging(),
			)

			// Create a shared cache for all handlers
			cache := NewCache(CacheTTL)
			// Create a shared sync.Map for thread-safe caching
			sharedCache := &sync.Map{}

			// Register tools and handlers
			registerNpmTool(s, cache, sharedCache)
			registerPythonTools(s, cache, sharedCache)
			registerJavaTools(s, cache, sharedCache)
			registerGoTool(s, cache, sharedCache)
			registerBedrockTools(s, cache, sharedCache)
			registerDockerTool(s, cache, sharedCache)
			registerSwiftTool(s, cache, sharedCache)

			// Start the server
			fmt.Fprintln(os.Stderr, "Starting MegaTool Package Version MCP Server...")
			if err := server.ServeStdio(s); err != nil {
				return fmt.Errorf("server error: %w", err)
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// registerNpmTool registers the npm version checking tool
func registerNpmTool(s *server.MCPServer, cache *Cache, sharedCache *sync.Map) {
	// Create NPM handler
	npmHandler := handlers.NewNpmHandler(nil, sharedCache)

	// Add NPM tool
	npmTool := mcp.NewTool("check_npm_versions",
		mcp.WithDescription("Check latest stable versions for npm packages"),
		mcp.WithObject("dependencies",
			mcp.Required(),
			mcp.Description("Dependencies object from package.json"),
		),
		mcp.WithObject("constraints",
			mcp.Description("Optional constraints for specific packages"),
		),
	)

	// Add NPM handler
	s.AddTool(npmTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return npmHandler.GetLatestVersion(ctx, request.Params.Arguments)
	})
}

// registerPythonTools registers the Python version checking tools
func registerPythonTools(s *server.MCPServer, cache *Cache, sharedCache *sync.Map) {
	// Create Python handler
	pythonHandler := handlers.NewPythonHandler(nil, sharedCache)

	// Tool for requirements.txt
	pythonTool := mcp.NewTool("check_python_versions",
		mcp.WithDescription("Check latest stable versions for Python packages"),
		mcp.WithArray("requirements",
			mcp.Required(),
			mcp.Description("Array of requirements from requirements.txt"),
		),
	)

	// Add Python requirements.txt handler
	s.AddTool(pythonTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return pythonHandler.GetLatestVersionFromRequirements(ctx, request.Params.Arguments)
	})

	// Tool for pyproject.toml
	pyprojectTool := mcp.NewTool("check_pyproject_versions",
		mcp.WithDescription("Check latest stable versions for Python packages in pyproject.toml"),
		mcp.WithObject("dependencies",
			mcp.Required(),
			mcp.Description("Dependencies object from pyproject.toml"),
		),
	)

	// Add Python pyproject.toml handler
	s.AddTool(pyprojectTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return pythonHandler.GetLatestVersionFromPyProject(ctx, request.Params.Arguments)
	})
}

// registerJavaTools registers the Java version checking tools
func registerJavaTools(s *server.MCPServer, cache *Cache, sharedCache *sync.Map) {
	// Create Java handler
	javaHandler := handlers.NewJavaHandler(nil, sharedCache)

	// Tool for Maven
	mavenTool := mcp.NewTool("check_maven_versions",
		mcp.WithDescription("Check latest stable versions for Java packages in pom.xml"),
		mcp.WithArray("dependencies",
			mcp.Required(),
			mcp.Description("Array of Maven dependencies"),
		),
	)

	// Add Maven handler
	s.AddTool(mavenTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return javaHandler.GetLatestVersionFromMaven(ctx, request.Params.Arguments)
	})

	// Tool for Gradle
	gradleTool := mcp.NewTool("check_gradle_versions",
		mcp.WithDescription("Check latest stable versions for Java packages in build.gradle"),
		mcp.WithArray("dependencies",
			mcp.Required(),
			mcp.Description("Array of Gradle dependencies"),
		),
	)

	// Add Gradle handler
	s.AddTool(gradleTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return javaHandler.GetLatestVersionFromGradle(ctx, request.Params.Arguments)
	})
}

// registerGoTool registers the Go version checking tool
func registerGoTool(s *server.MCPServer, cache *Cache, sharedCache *sync.Map) {
	// Create Go handler
	goHandler := handlers.NewGoHandler(nil, sharedCache)

	goTool := mcp.NewTool("check_go_versions",
		mcp.WithDescription("Check latest stable versions for Go packages in go.mod"),
		mcp.WithObject("dependencies",
			mcp.Required(),
			mcp.Description("Dependencies from go.mod"),
		),
	)

	// Add Go handler
	s.AddTool(goTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return goHandler.GetLatestVersion(ctx, request.Params.Arguments)
	})
}

// registerBedrockTools registers the AWS Bedrock tools
func registerBedrockTools(s *server.MCPServer, cache *Cache, sharedCache *sync.Map) {
	// Create Bedrock handler
	bedrockHandler := handlers.NewBedrockHandler(nil, sharedCache)

	// Tool for searching Bedrock models
	bedrockTool := mcp.NewTool("check_bedrock_models",
		mcp.WithDescription("Search, list, and get information about Amazon Bedrock models"),
		mcp.WithString("action",
			mcp.Description("Action to perform: list all models, search for models, or get a specific model"),
			mcp.Enum("list", "search", "get"),
			mcp.DefaultString("list"),
		),
		mcp.WithString("query",
			mcp.Description("Search query for model name or ID (used with action: \"search\")"),
		),
		mcp.WithString("provider",
			mcp.Description("Filter by provider name (used with action: \"search\")"),
		),
		mcp.WithString("region",
			mcp.Description("Filter by AWS region (used with action: \"search\")"),
		),
		mcp.WithString("modelId",
			mcp.Description("Model ID to retrieve (used with action: \"get\")"),
		),
	)

	// Add Bedrock handler
	s.AddTool(bedrockTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return bedrockHandler.GetLatestVersion(ctx, request.Params.Arguments)
	})

	// Tool for getting the latest Claude Sonnet model
	sonnetTool := mcp.NewTool("get_latest_bedrock_model",
		mcp.WithDescription("Get the latest Claude Sonnet model from Amazon Bedrock (best for coding tasks)"),
	)

	// Add Bedrock Claude Sonnet handler
	s.AddTool(sonnetTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Set the action to get_latest_claude_sonnet to use the specialized method
		return bedrockHandler.GetLatestVersion(ctx, map[string]interface{}{
			"action": "get_latest_claude_sonnet",
		})
	})
}

// registerDockerTool registers the Docker version checking tool
func registerDockerTool(s *server.MCPServer, cache *Cache, sharedCache *sync.Map) {
	// Create Docker handler
	dockerHandler := handlers.NewDockerHandler(nil, sharedCache)

	dockerTool := mcp.NewTool("check_docker_tags",
		mcp.WithDescription("Check available tags for Docker container images from Docker Hub, GitHub Container Registry, or custom registries"),
		mcp.WithString("image",
			mcp.Required(),
			mcp.Description("Docker image name (e.g., \"nginx\", \"ubuntu\", \"ghcr.io/owner/repo\")"),
		),
		mcp.WithString("registry",
			mcp.Description("Registry to check (dockerhub, ghcr, or custom)"),
			mcp.Enum("dockerhub", "ghcr", "custom"),
			mcp.DefaultString("dockerhub"),
		),
		mcp.WithString("customRegistry",
			mcp.Description("URL for custom registry (required when registry is \"custom\")"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of tags to return"),
			mcp.DefaultNumber(10),
		),
		mcp.WithArray("filterTags",
			mcp.Description("Array of regex patterns to filter tags"),
		),
		mcp.WithBoolean("includeDigest",
			mcp.Description("Include image digest in results"),
			mcp.DefaultBool(false),
		),
	)

	// Add Docker handler
	s.AddTool(dockerTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return dockerHandler.GetLatestVersion(ctx, request.Params.Arguments)
	})
}

// registerSwiftTool registers the Swift version checking tool
func registerSwiftTool(s *server.MCPServer, cache *Cache, sharedCache *sync.Map) {
	// Create Swift handler
	swiftHandler := handlers.NewSwiftHandler(nil, sharedCache)

	swiftTool := mcp.NewTool("check_swift_versions",
		mcp.WithDescription("Check latest stable versions for Swift packages in Package.swift"),
		mcp.WithArray("dependencies",
			mcp.Required(),
			mcp.Description("Array of Swift package dependencies"),
		),
		mcp.WithObject("constraints",
			mcp.Description("Optional constraints for specific packages"),
		),
	)

	// Add Swift handler
	s.AddTool(swiftTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return swiftHandler.GetLatestVersion(ctx, request.Params.Arguments)
	})
}
