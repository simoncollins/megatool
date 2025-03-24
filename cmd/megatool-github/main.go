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
	"github.com/megatool/internal/logging"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

const (
	// GitHubAPIBaseURL is the base URL for the GitHub API
	GitHubAPIBaseURL = "https://api.github.com"
)

// logger is the global logger instance
var logger *logrus.Logger

func main() {
	// Initialize logger
	pid := os.Getpid()
	log, err := logging.NewLogger("github", pid)
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
	}).Info("Starting GitHub MCP server")

	app := &cli.App{
		Name:  "megatool-github",
		Usage: "MegaTool GitHub MCP Server",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "configure",
				Aliases: []string{"c"},
				Usage:   "Configure GitHub MCP server",
			},
		},
		Action: func(c *cli.Context) error {
			// Handle configuration mode
			if c.Bool("configure") {
				logger.Info("Starting GitHub configuration")
				if err := configureGitHub(); err != nil {
					logger.WithError(err).Error("Configuration failed")
					return fmt.Errorf("configuration failed: %w", err)
				}
				logger.Info("GitHub MCP server configured successfully")
				fmt.Println("GitHub MCP server configured successfully")
				return nil
			}

			// Load configuration
			_, err := config.Load("github")
			if err != nil {
				logger.WithError(err).Error("Failed to load configuration")
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				fmt.Fprintf(os.Stderr, "Run 'megatool github --configure' to configure the GitHub MCP server\n")
				return err
			}

			// Get GitHub PAT from keyring
			pat, err := config.GetSecure("github", "pat")
			if err != nil {
				logger.WithError(err).Error("Failed to retrieve GitHub PAT")
				fmt.Fprintf(os.Stderr, "Error retrieving GitHub PAT: %v\n", err)
				fmt.Fprintf(os.Stderr, "Run 'megatool github --configure' to reconfigure the GitHub MCP server\n")
				return err
			}

			logger.Info("Configuration loaded successfully")

			// Create a new MCP server
			s := server.NewMCPServer(
				"MegaTool GitHub",
				"1.0.0",
				server.WithResourceCapabilities(true, true),
				server.WithToolCapabilities(true),
				// Don't use the built-in logging since we have our own
				// server.WithLogging(),
			)

			logger.Info("Initialized MCP server")

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
					logger.WithField("uri", request.Params.URI).Error("Invalid repository URI format")
					return nil, fmt.Errorf("invalid repository URI format")
				}

				owner := matches[1]
				repo := matches[2]

				logger.WithFields(logrus.Fields{
					"owner": owner,
					"repo":  repo,
					"uri":   request.Params.URI,
				}).Info("Fetching repository information")

				// Make GitHub API request
				repoInfo, err := getRepositoryInfo(pat, owner, repo)
				if err != nil {
					logger.WithFields(logrus.Fields{
						"owner": owner,
						"repo":  repo,
						"error": err.Error(),
					}).Error("Failed to get repository information")
					return nil, err
				}

				logger.WithFields(logrus.Fields{
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
					logger.WithField("error", "query must be a string").Error("Invalid parameter")
					return mcp.NewToolResultError("query must be a string"), nil
				}

				var limit int = 10
				if limitVal, ok := request.Params.Arguments["limit"].(float64); ok {
					limit = int(limitVal)
				}

				logger.WithFields(logrus.Fields{
					"query": query,
					"limit": limit,
				}).Info("Searching repositories")

				// Search repositories
				results, err := searchRepositories(pat, query, limit)
				if err != nil {
					logger.WithFields(logrus.Fields{
						"query": query,
						"limit": limit,
						"error": err.Error(),
					}).Error("Failed to search repositories")
					return mcp.NewToolResultError(fmt.Sprintf("Failed to search repositories: %v", err)), nil
				}

				logger.WithFields(logrus.Fields{
					"query": query,
					"limit": limit,
				}).Info("Repository search completed successfully")

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
					logger.WithField("error", "username must be a string").Error("Invalid parameter")
					return mcp.NewToolResultError("username must be a string"), nil
				}

				logger.WithField("username", username).Info("Fetching user information")

				// Get user information
				userInfo, err := getUserInfo(pat, username)
				if err != nil {
					logger.WithFields(logrus.Fields{
						"username": username,
						"error":    err.Error(),
					}).Error("Failed to get user information")
					return mcp.NewToolResultError(fmt.Sprintf("Failed to get user information: %v", err)), nil
				}

				logger.WithField("username", username).Info("User information fetched successfully")

				// Return results
				return mcp.NewToolResultText(userInfo), nil
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
		logger.WithError(err).Error("Failed to read input")
		return fmt.Errorf("failed to read input: %w", err)
	}
	pat = strings.TrimSpace(pat)

	logger.Info("Validating GitHub PAT")

	// Validate PAT by making a test API call
	client := &http.Client{}
	req, err := http.NewRequest("GET", GitHubAPIBaseURL+"/user", nil)
	if err != nil {
		logger.WithError(err).Error("Failed to create request")
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "token "+pat)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		logger.WithError(err).Error("Failed to connect to GitHub API")
		return fmt.Errorf("failed to connect to GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.WithFields(logrus.Fields{
			"status": resp.StatusCode,
			"body":   string(body),
		}).Error("Invalid GitHub PAT")
		return fmt.Errorf("invalid GitHub PAT (status %d): %s", resp.StatusCode, body)
	}

	logger.Info("GitHub PAT validated successfully")

	// Store PAT in keyring
	err = config.StoreSecure("github", "pat", pat)
	if err != nil {
		logger.WithError(err).Error("Failed to store PAT in keyring")
		return fmt.Errorf("failed to store PAT in keyring: %w", err)
	}

	// Create and save config
	cfg := &config.Config{
		APIEndpoint: GitHubAPIBaseURL,
	}
	err = config.Save("github", cfg)
	if err != nil {
		logger.WithError(err).Error("Failed to save configuration")
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	logger.Info("GitHub configuration saved successfully")
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
		logger.WithError(err).Error("Failed to create request")
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "token "+pat)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		logger.WithError(err).Error("Failed to connect to GitHub API")
		return "", fmt.Errorf("failed to connect to GitHub API: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.WithError(err).Error("Failed to read response body")
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		logger.WithFields(logrus.Fields{
			"status": resp.StatusCode,
			"body":   string(body),
		}).Error("GitHub API error")
		return "", fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, body)
	}

	// Pretty-print JSON
	var data interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		logger.WithError(err).Error("Failed to parse JSON")
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		logger.WithError(err).Error("Failed to format JSON")
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
		logger.WithError(err).Error("Failed to create request")
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "token "+pat)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		logger.WithError(err).Error("Failed to connect to GitHub API")
		return "", fmt.Errorf("failed to connect to GitHub API: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.WithError(err).Error("Failed to read response body")
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		logger.WithFields(logrus.Fields{
			"status": resp.StatusCode,
			"body":   string(body),
		}).Error("GitHub API error")
		return "", fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, body)
	}

	// Pretty-print JSON
	var data interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		logger.WithError(err).Error("Failed to parse JSON")
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		logger.WithError(err).Error("Failed to format JSON")
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
		logger.WithError(err).Error("Failed to create request")
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "token "+pat)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		logger.WithError(err).Error("Failed to connect to GitHub API")
		return "", fmt.Errorf("failed to connect to GitHub API: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.WithError(err).Error("Failed to read response body")
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		logger.WithFields(logrus.Fields{
			"status": resp.StatusCode,
			"body":   string(body),
		}).Error("GitHub API error")
		return "", fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, body)
	}

	// Pretty-print JSON
	var data interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		logger.WithError(err).Error("Failed to parse JSON")
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		logger.WithError(err).Error("Failed to format JSON")
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}

	return string(prettyJSON), nil
}
