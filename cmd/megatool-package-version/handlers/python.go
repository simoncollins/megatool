package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
)

const (
	// PyPIRegistryURL is the base URL for the PyPI registry
	PyPIRegistryURL = "https://pypi.org/pypi"
)

// PythonHandler handles Python package version checking
type PythonHandler struct {
	client HTTPClient
	cache  *sync.Map
}

// NewPythonHandler creates a new Python handler
func NewPythonHandler(client HTTPClient, cache *sync.Map) *PythonHandler {
	if client == nil {
		client = DefaultHTTPClient
	}
	if cache == nil {
		cache = &sync.Map{}
	}
	return &PythonHandler{
		client: client,
		cache:  cache,
	}
}

// PyPIPackageInfo represents information about a PyPI package
type PyPIPackageInfo struct {
	Info struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"info"`
	Releases map[string][]struct {
		PackageType string `json:"packagetype"`
	} `json:"releases"`
}

// getPackageInfo gets information about a PyPI package
func (h *PythonHandler) getPackageInfo(packageName string) (*PyPIPackageInfo, error) {
	// Check cache first
	if cachedInfo, ok := h.cache.Load(packageName); ok {
		return cachedInfo.(*PyPIPackageInfo), nil
	}

	// Make request to PyPI registry
	url := fmt.Sprintf("%s/%s/json", PyPIRegistryURL, url.PathEscape(packageName))
	body, err := MakeRequest(h.client, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PyPI package %s: %w", packageName, err)
	}

	// Parse response
	var info PyPIPackageInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("failed to parse PyPI package info: %w", err)
	}

	// Cache the result
	h.cache.Store(packageName, &info)

	return &info, nil
}

// getPackageVersion gets the latest version of a PyPI package
func (h *PythonHandler) getPackageVersion(packageName, currentVersion, label string) (*PackageVersion, error) {
	// Get package info
	info, err := h.getPackageInfo(packageName)
	if err != nil {
		return nil, err
	}

	// Get latest version
	latestVersion := info.Info.Version
	if latestVersion == "" {
		return nil, fmt.Errorf("latest version not found for package %s", packageName)
	}

	// Create result
	name := packageName
	if label != "" {
		name = fmt.Sprintf("%s (%s)", packageName, label)
	}

	result := &PackageVersion{
		Name:          name,
		LatestVersion: latestVersion,
		Registry:      "pypi",
	}

	if currentVersion != "" {
		// Remove any comparison operators from the current version
		cleanVersion := CleanVersion(currentVersion)
		result.CurrentVersion = StringPtr(cleanVersion)
	}

	return result, nil
}

// parseRequirement parses a requirement string from requirements.txt
func (h *PythonHandler) parseRequirement(requirement string) (name string, version string, err error) {
	// Remove comments
	if idx := strings.IndexByte(requirement, '#'); idx != -1 {
		requirement = requirement[:idx]
	}

	// Trim whitespace
	requirement = strings.TrimSpace(requirement)
	if requirement == "" {
		return "", "", fmt.Errorf("empty requirement")
	}

	// Skip options (lines starting with -)
	if strings.HasPrefix(requirement, "-") {
		return "", "", fmt.Errorf("requirement is an option")
	}

	// Parse package name and version
	re := regexp.MustCompile(`^([A-Za-z0-9_.-]+)(?:\s*([<>=!~]+.*)?)?$`)
	matches := re.FindStringSubmatch(requirement)
	if len(matches) < 2 {
		return "", "", fmt.Errorf("invalid requirement format: %s", requirement)
	}

	name = matches[1]
	if len(matches) > 2 && matches[2] != "" {
		version = strings.TrimSpace(matches[2])
	}

	return name, version, nil
}

// GetLatestVersionFromRequirements gets the latest versions for Python packages from requirements.txt
func (h *PythonHandler) GetLatestVersionFromRequirements(ctx context.Context, args interface{}) (*mcp.CallToolResult, error) {
	// Parse arguments
	var params struct {
		Requirements []string `json:"requirements"`
	}

	// Convert args to JSON and back to ensure proper type conversion
	jsonData, err := json.Marshal(args)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal arguments: %v", err)), nil
	}

	if err := json.Unmarshal(jsonData, &params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
	}

	if params.Requirements == nil {
		return mcp.NewToolResultError("Requirements array is required"), nil
	}

	// Check versions for each package
	results := make([]*PackageVersion, 0, len(params.Requirements))
	for _, requirement := range params.Requirements {
		name, version, err := h.parseRequirement(requirement)
		if err != nil {
			continue
		}

		result, err := h.getPackageVersion(name, version, "")
		if err != nil {
			fmt.Printf("Error checking PyPI package %s: %v\n", name, err)
			continue
		}

		results = append(results, result)
	}

	// Return results
	return NewToolResultJSON(results)
}

// GetLatestVersionFromPyProject gets the latest versions for Python packages from pyproject.toml
func (h *PythonHandler) GetLatestVersionFromPyProject(ctx context.Context, args interface{}) (*mcp.CallToolResult, error) {
	// Parse arguments
	var params struct {
		Dependencies struct {
			Dependencies         map[string]string            `json:"dependencies"`
			OptionalDependencies map[string]map[string]string `json:"optional-dependencies"`
			DevDependencies      map[string]string            `json:"dev-dependencies"`
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

	if params.Dependencies.Dependencies == nil &&
		params.Dependencies.OptionalDependencies == nil &&
		params.Dependencies.DevDependencies == nil {
		return mcp.NewToolResultError("No dependencies found in pyproject.toml"), nil
	}

	// Check versions for each package
	results := make([]*PackageVersion, 0)

	// Process main dependencies
	if params.Dependencies.Dependencies != nil {
		for name, version := range params.Dependencies.Dependencies {
			result, err := h.getPackageVersion(name, version, "")
			if err != nil {
				fmt.Printf("Error checking PyPI package %s: %v\n", name, err)
				continue
			}
			results = append(results, result)
		}
	}

	// Process optional dependencies
	if params.Dependencies.OptionalDependencies != nil {
		for group, deps := range params.Dependencies.OptionalDependencies {
			for name, version := range deps {
				result, err := h.getPackageVersion(name, version, fmt.Sprintf("optional: %s", group))
				if err != nil {
					fmt.Printf("Error checking PyPI package %s: %v\n", name, err)
					continue
				}
				results = append(results, result)
			}
		}
	}

	// Process dev dependencies
	if params.Dependencies.DevDependencies != nil {
		for name, version := range params.Dependencies.DevDependencies {
			result, err := h.getPackageVersion(name, version, "dev")
			if err != nil {
				fmt.Printf("Error checking PyPI package %s: %v\n", name, err)
				continue
			}
			results = append(results, result)
		}
	}

	// Return results
	return NewToolResultJSON(results)
}
