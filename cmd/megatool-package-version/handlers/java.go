package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
)

const (
	// MavenCentralURL is the base URL for the Maven Central repository search API
	MavenCentralURL = "https://search.maven.org/solrsearch/select"
)

// JavaHandler handles Java package version checking
type JavaHandler struct {
	client HTTPClient
	cache  *sync.Map
}

// NewJavaHandler creates a new Java handler
func NewJavaHandler(client HTTPClient, cache *sync.Map) *JavaHandler {
	if client == nil {
		client = DefaultHTTPClient
	}
	if cache == nil {
		cache = &sync.Map{}
	}
	return &JavaHandler{
		client: client,
		cache:  cache,
	}
}

// MavenSearchResponse represents a response from the Maven Central search API
type MavenSearchResponse struct {
	Response struct {
		NumFound int `json:"numFound"`
		Docs     []struct {
			ID         string `json:"id"`
			GroupID    string `json:"g"`
			ArtifactID string `json:"a"`
			Version    string `json:"v"`
		} `json:"docs"`
	} `json:"response"`
}

// getPackageVersion gets the latest version of a Maven package
func (h *JavaHandler) getPackageVersion(groupID, artifactID, currentVersion, scope string) (*PackageVersion, error) {
	// Create cache key
	cacheKey := fmt.Sprintf("%s:%s", groupID, artifactID)

	// Check cache first
	if cachedInfo, ok := h.cache.Load(cacheKey); ok {
		info := cachedInfo.(map[string]string)
		latestVersion := info["latestVersion"]

		// Create result
		name := fmt.Sprintf("%s:%s", groupID, artifactID)
		if scope != "" {
			name = fmt.Sprintf("%s (%s)", name, scope)
		}

		result := &PackageVersion{
			Name:          name,
			LatestVersion: latestVersion,
			Registry:      "maven",
		}

		if currentVersion != "" {
			result.CurrentVersion = StringPtr(currentVersion)
		}

		return result, nil
	}

	// Build query
	query := fmt.Sprintf("g:%q AND a:%q", groupID, artifactID)
	params := url.Values{}
	params.Set("q", query)
	params.Set("core", "gav")
	params.Set("rows", "1")
	params.Set("wt", "json")

	// Make request to Maven Central
	url := fmt.Sprintf("%s?%s", MavenCentralURL, params.Encode())
	body, err := MakeRequest(h.client, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Maven package %s:%s: %w", groupID, artifactID, err)
	}

	// Parse response
	var response MavenSearchResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse Maven search response: %w", err)
	}

	if response.Response.NumFound == 0 || len(response.Response.Docs) == 0 {
		return nil, fmt.Errorf("package not found: %s:%s", groupID, artifactID)
	}

	// Get latest version
	latestVersion := response.Response.Docs[0].Version

	// Cache the result
	h.cache.Store(cacheKey, map[string]string{
		"latestVersion": latestVersion,
	})

	// Create result
	name := fmt.Sprintf("%s:%s", groupID, artifactID)
	if scope != "" {
		name = fmt.Sprintf("%s (%s)", name, scope)
	}

	result := &PackageVersion{
		Name:          name,
		LatestVersion: latestVersion,
		Registry:      "maven",
	}

	if currentVersion != "" {
		result.CurrentVersion = StringPtr(currentVersion)
	}

	return result, nil
}

// GetLatestVersionFromMaven gets the latest versions for Maven packages
func (h *JavaHandler) GetLatestVersionFromMaven(ctx context.Context, args interface{}) (*mcp.CallToolResult, error) {
	// Parse arguments
	var params struct {
		Dependencies []struct {
			GroupID    string `json:"groupId"`
			ArtifactID string `json:"artifactId"`
			Version    string `json:"version,omitempty"`
			Scope      string `json:"scope,omitempty"`
		} `json:"dependencies"`
	}

	// Convert args to JSON and back to ensure proper type conversion
	jsonData, err := json.Marshal(args)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal arguments: %v", err)), nil
	}

	if err := json.Unmarshal(jsonData, &params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
	}

	if params.Dependencies == nil {
		return mcp.NewToolResultError("Dependencies array is required"), nil
	}

	// Check versions for each package
	results := make([]*PackageVersion, 0, len(params.Dependencies))
	for _, dep := range params.Dependencies {
		if dep.GroupID == "" || dep.ArtifactID == "" {
			continue
		}

		result, err := h.getPackageVersion(dep.GroupID, dep.ArtifactID, dep.Version, dep.Scope)
		if err != nil {
			fmt.Printf("Error checking Maven package %s:%s: %v\n", dep.GroupID, dep.ArtifactID, err)
			continue
		}

		results = append(results, result)
	}

	// Return results
	return NewToolResultJSON(results)
}

// GetLatestVersionFromGradle gets the latest versions for Gradle packages
func (h *JavaHandler) GetLatestVersionFromGradle(ctx context.Context, args interface{}) (*mcp.CallToolResult, error) {
	// Parse arguments
	var params struct {
		Dependencies []struct {
			Configuration string `json:"configuration"`
			Group         string `json:"group"`
			Name          string `json:"name"`
			Version       string `json:"version,omitempty"`
		} `json:"dependencies"`
	}

	// Convert args to JSON and back to ensure proper type conversion
	jsonData, err := json.Marshal(args)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal arguments: %v", err)), nil
	}

	if err := json.Unmarshal(jsonData, &params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
	}

	if params.Dependencies == nil {
		return mcp.NewToolResultError("Dependencies array is required"), nil
	}

	// Check versions for each package
	results := make([]*PackageVersion, 0, len(params.Dependencies))
	for _, dep := range params.Dependencies {
		if dep.Group == "" || dep.Name == "" || dep.Configuration == "" {
			continue
		}

		result, err := h.getPackageVersion(dep.Group, dep.Name, dep.Version, dep.Configuration)
		if err != nil {
			fmt.Printf("Error checking Gradle package %s:%s: %v\n", dep.Group, dep.Name, err)
			continue
		}

		results = append(results, result)
	}

	// Return results
	return NewToolResultJSON(results)
}
