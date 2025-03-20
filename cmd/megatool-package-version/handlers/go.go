package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
)

const (
	// GoProxyURL is the base URL for the Go proxy
	GoProxyURL = "https://proxy.golang.org"
)

// GoHandler handles Go package version checking
type GoHandler struct {
	client HTTPClient
	cache  *sync.Map
}

// NewGoHandler creates a new Go handler
func NewGoHandler(client HTTPClient, cache *sync.Map) *GoHandler {
	if client == nil {
		client = DefaultHTTPClient
	}
	if cache == nil {
		cache = &sync.Map{}
	}
	return &GoHandler{
		client: client,
		cache:  cache,
	}
}

// getPackageVersions gets the available versions of a Go package
func (h *GoHandler) getPackageVersions(packagePath string) ([]string, error) {
	// Check cache first
	if cachedVersions, ok := h.cache.Load(packagePath); ok {
		return cachedVersions.([]string), nil
	}

	// Make request to Go proxy
	url := fmt.Sprintf("%s/%s/@v/list", GoProxyURL, url.PathEscape(packagePath))
	body, err := MakeRequest(h.client, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Go package %s: %w", packagePath, err)
	}

	// Parse response
	versions := strings.Split(strings.TrimSpace(string(body)), "\n")
	if len(versions) == 1 && versions[0] == "" {
		return []string{}, nil
	}

	// Cache the result
	h.cache.Store(packagePath, versions)

	return versions, nil
}

// getPackageVersion gets the latest version of a Go package
func (h *GoHandler) getPackageVersion(packagePath, currentVersion string) (*PackageVersion, error) {
	// Get package versions
	versions, err := h.getPackageVersions(packagePath)
	if err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions found for package %s", packagePath)
	}

	// Get latest version (last in the list)
	latestVersion := versions[len(versions)-1]

	// Create result
	result := &PackageVersion{
		Name:          packagePath,
		LatestVersion: latestVersion,
		Registry:      "go",
	}

	if currentVersion != "" {
		// Remove any 'v' prefix from the current version
		cleanVersion := strings.TrimPrefix(currentVersion, "v")
		result.CurrentVersion = StringPtr(cleanVersion)
	}

	return result, nil
}

// GetLatestVersion gets the latest versions for Go packages
func (h *GoHandler) GetLatestVersion(ctx context.Context, args interface{}) (*mcp.CallToolResult, error) {
	// Parse arguments
	var params struct {
		Dependencies struct {
			Module  string `json:"module"`
			Require []struct {
				Path    string `json:"path"`
				Version string `json:"version,omitempty"`
			} `json:"require,omitempty"`
			Replace []struct {
				Old     string `json:"old"`
				New     string `json:"new"`
				Version string `json:"version,omitempty"`
			} `json:"replace,omitempty"`
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

	if params.Dependencies.Module == "" {
		return mcp.NewToolResultError("Module name is required"), nil
	}

	// Check versions for each package
	results := make([]*PackageVersion, 0)

	// Process required dependencies
	if params.Dependencies.Require != nil {
		for _, dep := range params.Dependencies.Require {
			if dep.Path == "" {
				continue
			}

			result, err := h.getPackageVersion(dep.Path, dep.Version)
			if err != nil {
				fmt.Printf("Error checking Go package %s: %v\n", dep.Path, err)
				continue
			}

			results = append(results, result)
		}
	}

	// Process replaced dependencies
	if params.Dependencies.Replace != nil {
		for _, rep := range params.Dependencies.Replace {
			if rep.Old == "" || rep.New == "" {
				continue
			}

			result, err := h.getPackageVersion(rep.New, rep.Version)
			if err != nil {
				fmt.Printf("Error checking Go package %s: %v\n", rep.New, err)
				continue
			}

			// Update name to show replacement
			result.Name = fmt.Sprintf("%s (replaces %s)", rep.New, rep.Old)

			results = append(results, result)
		}
	}

	// Return results
	return NewToolResultJSON(results)
}
