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
	"github.com/spf13/pflag"
)

const (
	// GitHubAPIBaseURL is the base URL for the GitHub API
	GitHubAPIBaseURL = "https://api.github.com"
)

func main() {
	// Parse command line flags
	var configureMode bool
	var showHelp bool
	pflag.BoolVarP(&configureMode, "configure", "c", false, "Configure GitHub MCP server")
	pflag.BoolVarP(&showHelp, "help", "h", false, "Show help")
	pflag.Parse()

	// Show help if requested
	if showHelp {
		fmt.Println("MegaTool GitHub MCP Server")
		fmt.Println()
		fmt.Println("Usage: megatool github [flags]")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  --configure, -c    Configure the GitHub MCP server")
		fmt.Println("  --help, -h         Show this help message")
		os.Exit(0)
	}

	// Handle configuration mode
	if configureMode {
		if err := configureGitHub(); err != nil {
			fmt.Fprintf(os.Stderr, "Configuration failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("GitHub MCP server configured successfully")
		os.Exit(0)
	}

	// Load configuration
	_, err := config.Load("github")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run 'megatool github --configure' to configure the GitHub MCP server\n")
		os.Exit(1)
	}

	// Get GitHub PAT from keyring
	pat, err := config.GetSecure("github", "pat")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving GitHub PAT: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run 'megatool github --configure' to reconfigure the GitHub MCP server\n")
		os.Exit(1)
	}

	// Create a new MCP server
	s := server.NewMCPServer(
		"MegaTool GitHub",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	// Add resource template for repository information
	repoTemplate := mcp.NewResourceTemplate(
		"github://repos/{owner}/{repo}",
		"GitHub Repository Information",
		mcp.WithTemplateDescription("Information about a GitHub repository"),
		mcp.WithTemplateMIMEType("application/json"),
	)

	// Add resource template handler
	s.AddResourceTemplate(repoTemplate, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		// Extract owner and repo from URI
		regex := regexp.MustCompile(`github://repos/([^/]+)/([^/]+)`)
		matches := regex.FindStringSubmatch(request.Params.URI)
		if len(matches) != 3 {
			return nil, fmt.Errorf("invalid repository URI format")
		}

		owner := matches[1]
		repo := matches[2]

		// Make GitHub API request
		repoInfo, err := getRepositoryInfo(pat, owner, repo)
		if err != nil {
			return nil, err
		}

		// Return resource contents
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     repoInfo,
			},
		}, nil
	})

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
	s.AddTool(searchReposTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		query, ok := request.Params.Arguments["query"].(string)
		if !ok {
			return mcp.NewToolResultError("query must be a string"), nil
		}

		var limit int = 10
		if limitVal, ok := request.Params.Arguments["limit"].(float64); ok {
			limit = int(limitVal)
		}

		// Search repositories
		results, err := searchRepositories(pat, query, limit)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to search repositories: %v", err)), nil
		}

		// Return results
		return mcp.NewToolResultText(results), nil
	})

	// Add tool for getting user information
	userInfoTool := mcp.NewTool("get_user",
		mcp.WithDescription("Get information about a GitHub user"),
		mcp.WithString("username",
			mcp.Required(),
			mcp.Description("GitHub username"),
		),
	)

	// Add tool handler for getting user information
	s.AddTool(userInfoTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		username, ok := request.Params.Arguments["username"].(string)
		if !ok {
			return mcp.NewToolResultError("username must be a string"), nil
		}

		// Get user information
		userInfo, err := getUserInfo(pat, username)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get user information: %v", err)), nil
		}

		// Return results
		return mcp.NewToolResultText(userInfo), nil
	})

	// Start the server
	fmt.Fprintln(os.Stderr, "Starting MegaTool GitHub MCP Server...")
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

// configureGitHub prompts the user for GitHub configuration
func configureGitHub() error {
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
		return fmt.Errorf("failed to read input: %w", err)
	}
	pat = strings.TrimSpace(pat)

	// Validate PAT by making a test API call
	client := &http.Client{}
	req, err := http.NewRequest("GET", GitHubAPIBaseURL+"/user", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "token "+pat)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("invalid GitHub PAT (status %d): %s", resp.StatusCode, body)
	}

	// Store PAT in keyring
	err = config.StoreSecure("github", "pat", pat)
	if err != nil {
		return fmt.Errorf("failed to store PAT in keyring: %w", err)
	}

	// Create and save config
	cfg := &config.Config{
		APIEndpoint: GitHubAPIBaseURL,
	}
	err = config.Save("github", cfg)
	if err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	return nil
}

// getRepositoryInfo gets information about a GitHub repository
func getRepositoryInfo(pat, owner, repo string) (string, error) {
	// Create HTTP client
	client := &http.Client{}

	// Create request
	apiURL := fmt.Sprintf("%s/repos/%s/%s", GitHubAPIBaseURL, owner, repo)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "token "+pat)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to GitHub API: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, body)
	}

	// Pretty-print JSON
	var data interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}

	return string(prettyJSON), nil
}

// searchRepositories searches for GitHub repositories
func searchRepositories(pat, query string, limit int) (string, error) {
	// Create HTTP client
	client := &http.Client{}

	// Create request
	apiURL := fmt.Sprintf("%s/search/repositories?q=%s&per_page=%d", GitHubAPIBaseURL, url.QueryEscape(query), limit)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "token "+pat)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to GitHub API: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, body)
	}

	// Pretty-print JSON
	var data interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}

	return string(prettyJSON), nil
}

// getUserInfo gets information about a GitHub user
func getUserInfo(pat, username string) (string, error) {
	// Create HTTP client
	client := &http.Client{}

	// Create request
	apiURL := fmt.Sprintf("%s/users/%s", GitHubAPIBaseURL, username)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "token "+pat)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to GitHub API: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, body)
	}

	// Pretty-print JSON
	var data interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}

	return string(prettyJSON), nil
}
