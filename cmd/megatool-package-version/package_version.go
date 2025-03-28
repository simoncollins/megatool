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
	"github.com/megatool/internal/mcpserver"
	"github.com/sirupsen/logrus"
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

// PackageVersionServer implements the MCPServerHandler interface for the package version server
type PackageVersionServer struct {
	logger     *logrus.Logger
	cache      *Cache
	sharedCache *sync.Map
}

// NewPackageVersionServer creates a new package version server
func NewPackageVersionServer() *PackageVersionServer {
	return &PackageVersionServer{
		cache:      NewCache(CacheTTL),
		sharedCache: &sync.Map{},
	}
}

// Name returns the display name of the server
func (s *PackageVersionServer) Name() string {
	return "Package Version"
}

// Capabilities returns the server capabilities
func (s *PackageVersionServer) Capabilities() []server.ServerOption {
	return []server.ServerOption{
		server.WithToolCapabilities(true),
	}
}

// Initialize sets up the server
func (s *PackageVersionServer) Initialize(srv *server.MCPServer) error {
	// Set up the logger
	pid := os.Getpid()
	logger, err := mcpserver.SetupLogger("package-version", pid)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	s.logger = logger

	s.logger.WithFields(logrus.Fields{
		"pid": pid,
	}).Info("Starting package-version MCP server")

	s.logger.Info("Initializing package version handlers")

	// Register tools and handlers
	s.registerNpmTool(srv)
	s.registerPythonTools(srv)
	s.registerJavaTools(srv)
	s.registerGoTool(srv)
	s.registerBedrockTools(srv)
	s.registerDockerTool(srv)
	s.registerSwiftTool(srv)

	s.logger.Info("All handlers registered successfully")
	return nil
}

// registerNpmTool registers the npm version checking tool
func (s *PackageVersionServer) registerNpmTool(srv *server.MCPServer) {
	s.logger.Info("Registering NPM version checking tool")

	// Create NPM handler
	npmHandler := handlers.NewNpmHandler(s.logger, s.sharedCache)

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
	srv.AddTool(npmTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithField("tool", "check_npm_versions").Info("Received request")
		return npmHandler.GetLatestVersion(ctx, request.Params.Arguments)
	})
}

// registerPythonTools registers the Python version checking tools
func (s *PackageVersionServer) registerPythonTools(srv *server.MCPServer) {
	s.logger.Info("Registering Python version checking tools")

	// Create Python handler
	pythonHandler := handlers.NewPythonHandler(s.logger, s.sharedCache)

	// Tool for requirements.txt
	pythonTool := mcp.NewTool("check_python_versions",
		mcp.WithDescription("Check latest stable versions for Python packages"),
		mcp.WithArray("requirements",
			mcp.Required(),
			mcp.Description("Array of requirements from requirements.txt"),
		),
	)

	// Add Python requirements.txt handler
	srv.AddTool(pythonTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithField("tool", "check_python_versions").Info("Received request")
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
	srv.AddTool(pyprojectTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithField("tool", "check_pyproject_versions").Info("Received request")
		return pythonHandler.GetLatestVersionFromPyProject(ctx, request.Params.Arguments)
	})
}

// registerJavaTools registers the Java version checking tools
func (s *PackageVersionServer) registerJavaTools(srv *server.MCPServer) {
	s.logger.Info("Registering Java version checking tools")

	// Create Java handler
	javaHandler := handlers.NewJavaHandler(s.logger, s.sharedCache)

	// Tool for Maven
	mavenTool := mcp.NewTool("check_maven_versions",
		mcp.WithDescription("Check latest stable versions for Java packages in pom.xml"),
		mcp.WithArray("dependencies",
			mcp.Required(),
			mcp.Description("Array of Maven dependencies"),
		),
	)

	// Add Maven handler
	srv.AddTool(mavenTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithField("tool", "check_maven_versions").Info("Received request")
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
	srv.AddTool(gradleTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithField("tool", "check_gradle_versions").Info("Received request")
		return javaHandler.GetLatestVersionFromGradle(ctx, request.Params.Arguments)
	})
}

// registerGoTool registers the Go version checking tool
func (s *PackageVersionServer) registerGoTool(srv *server.MCPServer) {
	s.logger.Info("Registering Go version checking tool")

	// Create Go handler
	goHandler := handlers.NewGoHandler(s.logger, s.sharedCache)

	goTool := mcp.NewTool("check_go_versions",
		mcp.WithDescription("Check latest stable versions for Go packages in go.mod"),
		mcp.WithObject("dependencies",
			mcp.Required(),
			mcp.Description("Dependencies from go.mod"),
		),
	)

	// Add Go handler
	srv.AddTool(goTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithField("tool", "check_go_versions").Info("Received request")
		return goHandler.GetLatestVersion(ctx, request.Params.Arguments)
	})
}

// registerBedrockTools registers the AWS Bedrock tools
func (s *PackageVersionServer) registerBedrockTools(srv *server.MCPServer) {
	s.logger.Info("Registering AWS Bedrock tools")

	// Create Bedrock handler
	bedrockHandler := handlers.NewBedrockHandler(s.logger, s.sharedCache)

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
	srv.AddTool(bedrockTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithFields(logrus.Fields{
			"tool":   "check_bedrock_models",
			"action": request.Params.Arguments["action"],
		}).Info("Received request")
		return bedrockHandler.GetLatestVersion(ctx, request.Params.Arguments)
	})

	// Tool for getting the latest Claude Sonnet model
	sonnetTool := mcp.NewTool("get_latest_bedrock_model",
		mcp.WithDescription("Get the latest Claude Sonnet model from Amazon Bedrock (best for coding tasks)"),
	)

	// Add Bedrock Claude Sonnet handler
	srv.AddTool(sonnetTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithField("tool", "get_latest_bedrock_model").Info("Received request")
		// Set the action to get_latest_claude_sonnet to use the specialized method
		return bedrockHandler.GetLatestVersion(ctx, map[string]interface{}{
			"action": "get_latest_claude_sonnet",
		})
	})
}

// registerDockerTool registers the Docker version checking tool
func (s *PackageVersionServer) registerDockerTool(srv *server.MCPServer) {
	s.logger.Info("Registering Docker version checking tool")

	// Create Docker handler
	dockerHandler := handlers.NewDockerHandler(s.logger, s.sharedCache)

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
	srv.AddTool(dockerTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithFields(logrus.Fields{
			"tool":     "check_docker_tags",
			"image":    request.Params.Arguments["image"],
			"registry": request.Params.Arguments["registry"],
		}).Info("Received request")
		return dockerHandler.GetLatestVersion(ctx, request.Params.Arguments)
	})
}

// registerSwiftTool registers the Swift version checking tool
func (s *PackageVersionServer) registerSwiftTool(srv *server.MCPServer) {
	s.logger.Info("Registering Swift version checking tool")

	// Create Swift handler
	swiftHandler := handlers.NewSwiftHandler(s.logger, s.sharedCache)

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
	srv.AddTool(swiftTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.WithField("tool", "check_swift_versions").Info("Received request")
		return swiftHandler.GetLatestVersion(ctx, request.Params.Arguments)
	})
}
