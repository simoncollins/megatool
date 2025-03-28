package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/megatool/internal/config"
	"github.com/megatool/internal/mcpserver"
	"github.com/sirupsen/logrus"
)

const (
	// GitHubAPIBaseURL is the base URL for the GitHub API
	GitHubAPIBaseURL = "https://api.github.com"
)

// GitHubServer implements the MCPServerHandler interface for the GitHub server
type GitHubServer struct {
	logger *logrus.Logger
	pat    string
}

// NewGitHubServer creates a new GitHub server
func NewGitHubServer() *GitHubServer {
	return &GitHubServer{}
}

// Name returns the display name of the server
func (s *GitHubServer) Name() string {
	return "GitHub"
}

// Capabilities returns the server capabilities
func (s *GitHubServer) Capabilities() []server.ServerOption {
	return []server.ServerOption{
		server.WithResourceCapabilities(true, true),
		server.WithToolCapabilities(true),
	}
}

// Configure handles the configuration of the GitHub server
func (s *GitHubServer) Configure() error {
	fmt.Println("Configuring GitHub MCP Server")
	fmt.Println()
	fmt.Println("You need a GitHub Personal Access Token (PAT) with the following scopes:")
	fmt.Println("- repo (for private repositories)")
	fmt.Println("- read:user (for user information)")
	fmt.Println()
	fmt.Println("You can create a new token at: https://github.com/settings/tokens")
	fmt.Println()

	// Prompt for GitHub PAT
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter GitHub Personal Access Token: ")
	pat, err := reader.ReadString('\n')
	if err != nil {
		if s.logger != nil {
			s.logger.WithError(err).Error("Failed to read input")
		}
		return fmt.Errorf("failed to read input: %w", err)
	}
	pat = strings.TrimSpace(pat)

	if s.logger != nil {
		s.logger.Info("Validating GitHub PAT")
	} else {
		fmt.Println("Validating GitHub PAT...")
	}

	// Validate PAT by making a test API call
	client := &http.Client{}
	req, err := http.NewRequest("GET", GitHubAPIBaseURL+"/user", nil)
	if err != nil {
		if s.logger != nil {
			s.logger.WithError(err).Error("Failed to create request")
		}
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "token "+pat)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		if s.logger != nil {
			s.logger.WithError(err).Error("Failed to connect to GitHub API")
		}
		return fmt.Errorf("failed to connect to GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if s.logger != nil {
			s.logger.WithFields(logrus.Fields{
				"status": resp.StatusCode,
				"body":   string(body),
			}).Error("Invalid GitHub PAT")
		}
		return fmt.Errorf("invalid GitHub PAT (status %d): %s", resp.StatusCode, body)
	}

	if s.logger != nil {
		s.logger.Info("GitHub PAT validated successfully")
	} else {
		fmt.Println("GitHub PAT validated successfully")
	}

	// Store PAT in keyring
	err = config.StoreSecure("github", "pat", pat)
	if err != nil {
		if s.logger != nil {
			s.logger.WithError(err).Error("Failed to store PAT in keyring")
		}
		return fmt.Errorf("failed to store PAT in keyring: %w", err)
	}

	// Create and save config
	cfg := &config.Config{
		APIEndpoint: GitHubAPIBaseURL,
	}
	err = config.Save("github", cfg)
	if err != nil {
		if s.logger != nil {
			s.logger.WithError(err).Error("Failed to save configuration")
		}
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("GitHub configuration saved successfully")
	} else {
		fmt.Println("GitHub configuration saved successfully")
	}
	return nil
}

// LoadConfig loads the GitHub server configuration
func (s *GitHubServer) LoadConfig() error {
	// Load configuration
	_, err := config.Load("github")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get GitHub PAT from keyring
	pat, err := config.GetSecure("github", "pat")
	if err != nil {
		return fmt.Errorf("failed to retrieve GitHub PAT: %w", err)
	}

	s.pat = pat
	return nil
}

// Initialize sets up the server
func (s *GitHubServer) Initialize(srv *server.MCPServer) error {
	// Set up the logger
	pid := os.Getpid()
	logger, err := mcpserver.SetupLogger("github", pid)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	s.logger = logger

	s.logger.WithFields(logrus.Fields{
		"pid": pid,
	}).Info("Starting GitHub MCP server")

	// Register resource template
	s.registerRepositoryResource(srv)

	// Register tools
	s.registerSearchReposTool(srv)
	s.registerUserInfoTool(srv)

	s.logger.Info("GitHub server initialized")
	return nil
}

// registerRepositoryResource registers the repository resource template
func (s *GitHubServer) registerRepositoryResource(srv *server.MCPServer) {
	// Add resource template for repository information
	repoTemplate := mcp.NewResourceTemplate(
		"github://repos/{owner}/{repo}",
		"GitHub Repository Information",
		mcp.WithTemplateDescription("Information about a GitHub repository"),
		mcp.WithTemplateMIMEType("application/json"),
	)

	// Add resource template handler
	srv.AddResourceTemplate(repoTemplate, s.handleRepositoryResource)
}

// handleRepositoryResource handles the repository resource template
func (s *GitHubServer) handleRepositoryResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract owner and repo from URI
	regex := regexp.MustCompile(`github://repos/([^/]+)/([^/]+)`)
	matches := regex.FindStringSubmatch(request.Params.URI)
	if len(matches) != 3 {
		s.logger.WithField("uri", request.Params.URI).Error("Invalid repository URI format")
		return nil, fmt.Errorf("invalid repository URI format")
	}

	owner := matches[1]
	repo := matches[2]

	s.logger.WithFields(logrus.Fields{
		"owner": owner,
		"repo":  repo,
		"uri":   request.Params.URI,
	}).Info("Fetching repository information")

	// Make GitHub API request
	repoInfo, err := s.getRepositoryInfo(owner, repo)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"owner": owner,
			"repo":  repo,
			"error": err.Error(),
		}).Error("Failed to get repository information")
		return nil, err
	}

	s.logger.WithFields(logrus.Fields{
		"owner": owner,
		"repo":  repo,
	}).Info("Repository information fetched successfully")

	// Return resource contents
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     repoInfo,
		},
	}, nil
}

// registerSearchReposTool registers the search repositories tool
func (s *GitHubServer) registerSearchReposTool(srv *server.MCPServer) {
	// Add tool for searching repositories
	searchReposTool := mcp.NewTool("search_repos",
		mcp.WithDescription("Search for GitHub repositories"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results to return"),
		),
	)

	// Add tool handler for searching repositories
	srv.AddTool(searchReposTool, s.handleSearchRepos)
}

// handleSearchRepos handles the search repositories tool
func (s *GitHubServer) handleSearchRepos(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcpserver.HandleToolRequest(ctx, request, s.performSearchRepos, s.logger)
}

// performSearchRepos performs the repository search
func (s *GitHubServer) performSearchRepos(ctx context.Context, args map[string]interface{}) (string, error) {
	// Extract parameters
	query, ok := mcpserver.ExtractStringParam(args, "query", s.logger)
	if !ok {
		return "", fmt.Errorf("query must be a string")
	}

	var limit int = 10
	if limitVal, ok := args["limit"].(float64); ok {
		limit = int(limitVal)
	}

	s.logger.WithFields(logrus.Fields{
		"query": query,
		"limit": limit,
	}).Info("Searching repositories")

	// Search repositories
	results, err := s.searchRepositories(query, limit)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"query": query,
			"limit": limit,
			"error": err.Error(),
		}).Error("Failed to search repositories")
		return "", fmt.Errorf("failed to search repositories: %v", err)
	}

	s.logger.WithFields(logrus.Fields{
		"query": query,
		"limit": limit,
	}).Info("Repository search completed successfully")

	return results, nil
}

// registerUserInfoTool registers the user information tool
func (s *GitHubServer) registerUserInfoTool(srv *server.MCPServer) {
	// Add tool for getting user information
	userInfoTool := mcp.NewTool("get_user",
		mcp.WithDescription("Get information about a GitHub user"),
		mcp.WithString("username",
			mcp.Required(),
			mcp.Description("GitHub username"),
		),
	)

	// Add tool handler for getting user information
	srv.AddTool(userInfoTool, s.handleUserInfo)
}

// handleUserInfo handles the user information tool
func (s *GitHubServer) handleUserInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcpserver.HandleToolRequest(ctx, request, s.performGetUserInfo, s.logger)
}

// performGetUserInfo performs the user information retrieval
func (s *GitHubServer) performGetUserInfo(ctx context.Context, args map[string]interface{}) (string, error) {
	// Extract parameters
	username, ok := mcpserver.ExtractStringParam(args, "username", s.logger)
	if !ok {
		return "", fmt.Errorf("username must be a string")
	}

	s.logger.WithField("username", username).Info("Fetching user information")

	// Get user information
	userInfo, err := s.getUserInfo(username)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"username": username,
			"error":    err.Error(),
		}).Error("Failed to get user information")
		return "", fmt.Errorf("failed to get user information: %v", err)
	}

	s.logger.WithField("username", username).Info("User information fetched successfully")

	return userInfo, nil
}

// getRepositoryInfo gets information about a GitHub repository
func (s *GitHubServer) getRepositoryInfo(owner, repo string) (string, error) {
	// Create HTTP client
	client := &http.Client{}

	// Create request
	apiURL := fmt.Sprintf("%s/repos/%s/%s", GitHubAPIBaseURL, owner, repo)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create request")
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "token "+s.pat)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		s.logger.WithError(err).Error("Failed to connect to GitHub API")
		return "", fmt.Errorf("failed to connect to GitHub API: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.WithError(err).Error("Failed to read response body")
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		s.logger.WithFields(logrus.Fields{
			"status": resp.StatusCode,
			"body":   string(body),
		}).Error("GitHub API error")
		return "", fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, body)
	}

	// Pretty-print JSON
	var data interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		s.logger.WithError(err).Error("Failed to parse JSON")
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		s.logger.WithError(err).Error("Failed to format JSON")
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}

	return string(prettyJSON), nil
}

// searchRepositories searches for GitHub repositories
func (s *GitHubServer) searchRepositories(query string, limit int) (string, error) {
	// Create HTTP client
	client := &http.Client{}

	// Create request
	apiURL := fmt.Sprintf("%s/search/repositories?q=%s&per_page=%d", GitHubAPIBaseURL, url.QueryEscape(query), limit)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create request")
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "token "+s.pat)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		s.logger.WithError(err).Error("Failed to connect to GitHub API")
		return "", fmt.Errorf("failed to connect to GitHub API: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.WithError(err).Error("Failed to read response body")
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		s.logger.WithFields(logrus.Fields{
			"status": resp.StatusCode,
			"body":   string(body),
		}).Error("GitHub API error")
		return "", fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, body)
	}

	// Pretty-print JSON
	var data interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		s.logger.WithError(err).Error("Failed to parse JSON")
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		s.logger.WithError(err).Error("Failed to format JSON")
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}

	return string(prettyJSON), nil
}

// getUserInfo gets information about a GitHub user
func (s *GitHubServer) getUserInfo(username string) (string, error) {
	// Create HTTP client
	client := &http.Client{}

	// Create request
	apiURL := fmt.Sprintf("%s/users/%s", GitHubAPIBaseURL, username)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create request")
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "token "+s.pat)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		s.logger.WithError(err).Error("Failed to connect to GitHub API")
		return "", fmt.Errorf("failed to connect to GitHub API: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.WithError(err).Error("Failed to read response body")
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		s.logger.WithFields(logrus.Fields{
			"status": resp.StatusCode,
			"body":   string(body),
		}).Error("GitHub API error")
		return "", fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, body)
	}

	// Pretty-print JSON
	var data interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		s.logger.WithError(err).Error("Failed to parse JSON")
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		s.logger.WithError(err).Error("Failed to format JSON")
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}

	return string(prettyJSON), nil
}
