package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
)

const (
	// NpmRegistryURL is the base URL for the npm registry
	NpmRegistryURL = "https://registry.npmjs.org"
)

// NpmHandler handles npm package version checking
type NpmHandler struct {
	client HTTPClient
	cache  *sync.Map
}

// NewNpmHandler creates a new npm handler
func NewNpmHandler(client HTTPClient, cache *sync.Map) *NpmHandler {
	if client == nil {
		client = DefaultHTTPClient
	}
	if cache == nil {
		cache = &sync.Map{}
	}
	return &NpmHandler{
		client: client,
		cache:  cache,
	}
}

// NpmPackageInfo represents information about an npm package
type NpmPackageInfo struct {
	Name     string            `json:"name"`
	DistTags map[string]string `json:"dist-tags"`
	Versions map[string]struct {
		Version string `json:"version"`
	} `json:"versions"`
}

// getPackageInfo gets information about an npm package
func (h *NpmHandler) getPackageInfo(packageName string) (*NpmPackageInfo, error) {
	// Check cache first
	if cachedInfo, ok := h.cache.Load(packageName); ok {
		return cachedInfo.(*NpmPackageInfo), nil
	}

	// Make request to npm registry
	url := fmt.Sprintf("%s/%s", NpmRegistryURL, url.PathEscape(packageName))
	body, err := MakeRequest(h.client, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch npm package %s: %w", packageName, err)
	}

	// Parse response
	var info NpmPackageInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("failed to parse npm package info: %w", err)
	}

	// Cache the result
	h.cache.Store(packageName, &info)

	return &info, nil
}

// getPackageVersion gets the latest version of an npm package
func (h *NpmHandler) getPackageVersion(packageName, currentVersion string, constraint *VersionConstraint) (*PackageVersion, error) {
	// Check if package should be excluded
	if constraint != nil && constraint.ExcludePackage {
		cleanVersion := CleanVersion(currentVersion)
		return &PackageVersion{
			Name:           packageName,
			CurrentVersion: StringPtr(cleanVersion),
			LatestVersion:  cleanVersion,
			Registry:       "npm",
			Skipped:        true,
			SkipReason:     "Package excluded from updates",
		}, nil
	}

	// Get package info
	info, err := h.getPackageInfo(packageName)
	if err != nil {
		return nil, err
	}

	// Get latest version
	latestVersion := info.DistTags["latest"]
	if latestVersion == "" {
		return nil, fmt.Errorf("latest version not found for package %s", packageName)
	}

	// If major version constraint exists, find the latest version within that major
	if constraint != nil && constraint.MajorVersion != nil {
		targetMajor := *constraint.MajorVersion
		versions := make([]string, 0, len(info.Versions))
		for version := range info.Versions {
			major, _, _, err := ParseVersion(version)
			if err != nil {
				continue
			}
			if major == targetMajor {
				versions = append(versions, version)
			}
		}

		if len(versions) > 0 {
			// Sort versions
			sort.Slice(versions, func(i, j int) bool {
				cmp, err := CompareVersions(versions[i], versions[j])
				if err != nil {
					return false
				}
				return cmp > 0
			})
			latestVersion = versions[0]
		}
	}

	// Create result
	result := &PackageVersion{
		Name:          packageName,
		LatestVersion: latestVersion,
		Registry:      "npm",
	}

	if currentVersion != "" {
		// Remove any leading ^ or ~ from the current version
		cleanVersion := CleanVersion(currentVersion)
		result.CurrentVersion = StringPtr(cleanVersion)
	}

	if constraint != nil && constraint.MajorVersion != nil {
		result.SkipReason = fmt.Sprintf("Limited to major version %d", *constraint.MajorVersion)
	}

	return result, nil
}

// GetLatestVersion gets the latest versions for npm packages
func (h *NpmHandler) GetLatestVersion(ctx context.Context, args interface{}) (*mcp.CallToolResult, error) {
	// Parse arguments
	var params struct {
		Dependencies map[string]string      `json:"dependencies"`
		Constraints  map[string]interface{} `json:"constraints"`
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
		return mcp.NewToolResultError("Dependencies object is required"), nil
	}

	// Parse constraints
	constraints := make(map[string]*VersionConstraint)
	if params.Constraints != nil {
		for pkg, c := range params.Constraints {
			constraintData, err := json.Marshal(c)
			if err != nil {
				continue
			}

			var constraint VersionConstraint
			if err := json.Unmarshal(constraintData, &constraint); err != nil {
				continue
			}

			constraints[pkg] = &constraint
		}
	}

	// Check versions for each package
	results := make([]*PackageVersion, 0, len(params.Dependencies))
	for name, version := range params.Dependencies {
		if strings.TrimSpace(version) == "" {
			continue
		}

		var constraint *VersionConstraint
		if c, ok := constraints[name]; ok {
			constraint = c
		}

		result, err := h.getPackageVersion(name, version, constraint)
		if err != nil {
			fmt.Printf("Error checking npm package %s: %v\n", name, err)
			continue
		}

		results = append(results, result)
	}

	// Return results
	return NewToolResultJSON(results)
}
