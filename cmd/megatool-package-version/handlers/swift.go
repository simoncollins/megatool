package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
)

const (
	// GitHubAPIURL is the base URL for the GitHub API
	GitHubAPIURL = "https://api.github.com"
)

// SwiftHandler handles Swift package version checking
type SwiftHandler struct {
	client HTTPClient
	cache  *sync.Map
}

// NewSwiftHandler creates a new Swift handler
func NewSwiftHandler(client HTTPClient, cache *sync.Map) *SwiftHandler {
	if client == nil {
		client = DefaultHTTPClient
	}
	if cache == nil {
		cache = &sync.Map{}
	}
	return &SwiftHandler{
		client: client,
		cache:  cache,
	}
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Prerelease bool   `json:"prerelease"`
	CreatedAt  string `json:"created_at"`
}

// extractGitHubInfo extracts owner and repo from a GitHub URL
func (h *SwiftHandler) extractGitHubInfo(packageURL string) (owner, repo string, isGitHub bool) {
	// Check if it's a GitHub URL
	re := regexp.MustCompile(`github\.com[/:]([^/]+)/([^/]+?)(?:\.git)?$`)
	matches := re.FindStringSubmatch(packageURL)
	if len(matches) != 3 {
		return "", "", false
	}
	return matches[1], matches[2], true
}

// getGitHubReleases gets the releases for a GitHub repository
func (h *SwiftHandler) getGitHubReleases(owner, repo string) ([]GitHubRelease, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("github-releases:%s/%s", owner, repo)
	if cachedReleases, ok := h.cache.Load(cacheKey); ok {
		return cachedReleases.([]GitHubRelease), nil
	}

	// Build URL
	releasesURL := fmt.Sprintf("%s/repos/%s/%s/releases", GitHubAPIURL, owner, repo)

	// Set headers
	headers := make(map[string]string)
	if token := h.getGitHubToken(); token != "" {
		headers["Authorization"] = "token " + token
	}

	// Make request
	body, err := MakeRequest(h.client, "GET", releasesURL, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub releases: %w", err)
	}

	// Parse response
	var releases []GitHubRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub releases response: %w", err)
	}

	// Cache the result
	h.cache.Store(cacheKey, releases)

	return releases, nil
}

// getGitHubToken gets a GitHub token from environment variables
func (h *SwiftHandler) getGitHubToken() string {
	// GitHub token would typically be provided via environment variables
	return "" // TODO: Implement token retrieval if needed
}

// getPackageVersion gets the latest version of a Swift package
func (h *SwiftHandler) getPackageVersion(packageURL, currentVersion, requirement string, constraint *VersionConstraint) (*PackageVersion, error) {
	// Extract package name from URL
	packageName := packageURL
	if idx := strings.LastIndex(packageURL, "/"); idx != -1 {
		packageName = packageURL[idx+1:]
	}
	packageName = strings.TrimSuffix(packageName, ".git")

	// Check if package should be excluded
	if constraint != nil && constraint.ExcludePackage {
		return &PackageVersion{
			Name:           packageName,
			CurrentVersion: StringPtr(currentVersion),
			LatestVersion:  currentVersion,
			Registry:       "swift",
			Skipped:        true,
			SkipReason:     "Package excluded from updates",
		}, nil
	}

	// For GitHub repositories, we can use the GitHub API to get the latest release
	owner, repo, isGitHub := h.extractGitHubInfo(packageURL)
	if !isGitHub {
		// For non-GitHub repositories, we can't easily determine the latest version
		return &PackageVersion{
			Name:           packageName,
			CurrentVersion: StringPtr(currentVersion),
			LatestVersion:  currentVersion,
			Registry:       "swift",
			Skipped:        true,
			SkipReason:     "Non-GitHub repository, cannot determine latest version",
		}, nil
	}

	// Get releases
	releases, err := h.getGitHubReleases(owner, repo)
	if err != nil {
		return nil, err
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found for package %s", packageName)
	}

	// Find the latest release that's not a pre-release
	var latestRelease *GitHubRelease
	for i := range releases {
		if !releases[i].Prerelease {
			latestRelease = &releases[i]
			break
		}
	}

	// If no stable release is found, use the latest release
	if latestRelease == nil {
		latestRelease = &releases[0]
	}

	// Get latest version
	latestVersion := latestRelease.TagName
	if strings.HasPrefix(latestVersion, "v") {
		latestVersion = latestVersion[1:]
	}

	// If major version constraint exists, check if the latest version complies
	if constraint != nil && constraint.MajorVersion != nil {
		targetMajor := *constraint.MajorVersion
		constrainedReleases := make([]GitHubRelease, 0)

		for _, release := range releases {
			version := release.TagName
			if strings.HasPrefix(version, "v") {
				version = version[1:]
			}

			major, _, _, err := ParseVersion(version)
			if err != nil {
				continue
			}

			if major == targetMajor {
				constrainedReleases = append(constrainedReleases, release)
			}
		}

		if len(constrainedReleases) > 0 {
			// Sort releases by version (higher versions first)
			sort.Slice(constrainedReleases, func(i, j int) bool {
				vi := constrainedReleases[i].TagName
				vj := constrainedReleases[j].TagName
				if strings.HasPrefix(vi, "v") {
					vi = vi[1:]
				}
				if strings.HasPrefix(vj, "v") {
					vj = vj[1:]
				}
				cmp, err := CompareVersions(vi, vj)
				if err != nil {
					return false
				}
				return cmp > 0
			})

			// Use the highest version that matches the constraint
			latestVersion = constrainedReleases[0].TagName
			if strings.HasPrefix(latestVersion, "v") {
				latestVersion = latestVersion[1:]
			}
		}
	}

	// Create result
	result := &PackageVersion{
		Name:          packageName,
		LatestVersion: latestVersion,
		Registry:      "swift",
	}

	if currentVersion != "" {
		result.CurrentVersion = StringPtr(currentVersion)
	}

	if constraint != nil && constraint.MajorVersion != nil {
		result.SkipReason = fmt.Sprintf("Limited to major version %d", *constraint.MajorVersion)
	}

	return result, nil
}

// GetLatestVersion gets the latest versions for Swift packages
func (h *SwiftHandler) GetLatestVersion(ctx context.Context, args interface{}) (*mcp.CallToolResult, error) {
	// Parse arguments
	var params struct {
		Dependencies []struct {
			URL         string `json:"url"`
			Version     string `json:"version,omitempty"`
			Requirement string `json:"requirement,omitempty"`
		} `json:"dependencies"`
		Constraints map[string]interface{} `json:"constraints,omitempty"`
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
	for _, dep := range params.Dependencies {
		if dep.URL == "" {
			continue
		}

		var constraint *VersionConstraint
		if c, ok := constraints[dep.URL]; ok {
			constraint = c
		}

		result, err := h.getPackageVersion(dep.URL, dep.Version, dep.Requirement, constraint)
		if err != nil {
			fmt.Printf("Error checking Swift package %s: %v\n", dep.URL, err)
			continue
		}

		results = append(results, result)
	}

	// Return results
	return NewToolResultJSON(results)
}
