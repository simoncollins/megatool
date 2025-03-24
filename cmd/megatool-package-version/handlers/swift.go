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
	"github.com/sirupsen/logrus"
)

const (
	// GitHubAPIURL is the base URL for the GitHub API
	GitHubAPIURL = "https://api.github.com"
)

// SwiftHandler handles Swift package version checking
type SwiftHandler struct {
	client HTTPClient
	cache  *sync.Map
	logger *logrus.Logger
}

// NewSwiftHandler creates a new Swift handler
func NewSwiftHandler(logger *logrus.Logger, cache *sync.Map) *SwiftHandler {
	if cache == nil {
		cache = &sync.Map{}
	}
	return &SwiftHandler{
		client: DefaultHTTPClient,
		cache:  cache,
		logger: logger,
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
	if h.logger != nil {
		h.logger.WithField("packageURL", packageURL).Debug("Extracting GitHub info from URL")
	}

	// Check if it's a GitHub URL
	re := regexp.MustCompile(`github\.com[/:]([^/]+)/([^/]+?)(?:\.git)?$`)
	matches := re.FindStringSubmatch(packageURL)
	if len(matches) != 3 {
		if h.logger != nil {
			h.logger.WithField("packageURL", packageURL).Debug("Not a GitHub URL")
		}
		return "", "", false
	}

	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"packageURL": packageURL,
			"owner":      matches[1],
			"repo":       matches[2],
		}).Debug("Successfully extracted GitHub info")
	}

	return matches[1], matches[2], true
}

// getGitHubReleases gets the releases for a GitHub repository
func (h *SwiftHandler) getGitHubReleases(owner, repo string) ([]GitHubRelease, error) {
	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"owner": owner,
			"repo":  repo,
		}).Debug("Getting GitHub releases")
	}

	// Check cache first
	cacheKey := fmt.Sprintf("github-releases:%s/%s", owner, repo)
	if cachedReleases, ok := h.cache.Load(cacheKey); ok {
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"owner":       owner,
				"repo":        repo,
				"releaseCount": len(cachedReleases.([]GitHubRelease)),
			}).Debug("Using cached GitHub releases")
		}
		return cachedReleases.([]GitHubRelease), nil
	}

	// Build URL
	releasesURL := fmt.Sprintf("%s/repos/%s/%s/releases", GitHubAPIURL, owner, repo)

	// Set headers
	headers := make(map[string]string)
	if token := h.getGitHubToken(); token != "" {
		headers["Authorization"] = "token " + token
	}

	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"url":     releasesURL,
			"hasAuth": len(headers) > 0,
		}).Debug("Making GitHub API request")
	}

	// Make request
	body, err := MakeRequestWithLogger(h.client, h.logger, "GET", releasesURL, headers)
	if err != nil {
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"owner": owner,
				"repo":  repo,
				"error": err.Error(),
			}).Error("Failed to get GitHub releases")
		}
		return nil, fmt.Errorf("failed to get GitHub releases: %w", err)
	}

	// Parse response
	var releases []GitHubRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"owner": owner,
				"repo":  repo,
				"error": err.Error(),
			}).Error("Failed to parse GitHub releases response")
		}
		return nil, fmt.Errorf("failed to parse GitHub releases response: %w", err)
	}

	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"owner":       owner,
			"repo":        repo,
			"releaseCount": len(releases),
		}).Debug("Successfully got GitHub releases")
	}

	// Cache the result
	h.cache.Store(cacheKey, releases)

	return releases, nil
}

// getGitHubToken gets a GitHub token from environment variables
func (h *SwiftHandler) getGitHubToken() string {
	if h.logger != nil {
		h.logger.Debug("Getting GitHub token")
	}

	// GitHub token would typically be provided via environment variables
	return "" // TODO: Implement token retrieval if needed
}

// getPackageVersion gets the latest version of a Swift package
func (h *SwiftHandler) getPackageVersion(packageURL, currentVersion, requirement string, constraint *VersionConstraint) (*PackageVersion, error) {
	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"packageURL":     packageURL,
			"currentVersion": currentVersion,
			"requirement":    requirement,
			"hasConstraint":  constraint != nil,
		}).Debug("Getting Swift package version")
	}

	// Extract package name from URL
	packageName := packageURL
	if idx := strings.LastIndex(packageURL, "/"); idx != -1 {
		packageName = packageURL[idx+1:]
	}
	packageName = strings.TrimSuffix(packageName, ".git")

	// Check if package should be excluded
	if constraint != nil && constraint.ExcludePackage {
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"packageURL": packageURL,
				"packageName": packageName,
			}).Debug("Package excluded from updates")
		}
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
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"packageURL":  packageURL,
				"packageName": packageName,
			}).Debug("Non-GitHub repository, cannot determine latest version")
		}
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
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"packageURL":  packageURL,
				"packageName": packageName,
				"owner":       owner,
				"repo":        repo,
			}).Error("No releases found for package")
		}
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
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"packageURL":  packageURL,
				"packageName": packageName,
			}).Debug("No stable release found, using latest release")
		}
		latestRelease = &releases[0]
	}

	// Get latest version
	latestVersion := latestRelease.TagName
	if strings.HasPrefix(latestVersion, "v") {
		latestVersion = latestVersion[1:]
	}

	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"packageURL":    packageURL,
			"packageName":   packageName,
			"latestVersion": latestVersion,
		}).Debug("Found latest version")
	}

	// If major version constraint exists, check if the latest version complies
	if constraint != nil && constraint.MajorVersion != nil {
		targetMajor := *constraint.MajorVersion
		constrainedReleases := make([]GitHubRelease, 0)

		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"packageURL":   packageURL,
				"packageName":  packageName,
				"majorVersion": targetMajor,
			}).Debug("Applying major version constraint")
		}

		for _, release := range releases {
			version := release.TagName
			if strings.HasPrefix(version, "v") {
				version = version[1:]
			}

			major, _, _, err := ParseVersion(version)
			if err != nil {
				if h.logger != nil {
					h.logger.WithFields(logrus.Fields{
						"version": version,
						"error":   err.Error(),
					}).Debug("Failed to parse version")
				}
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

			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"packageURL":    packageURL,
					"packageName":   packageName,
					"majorVersion":  targetMajor,
					"latestVersion": latestVersion,
				}).Debug("Found latest version matching major version constraint")
			}
		} else {
			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"packageURL":   packageURL,
					"packageName":  packageName,
					"majorVersion": targetMajor,
				}).Warn("No releases found matching major version constraint")
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
	if h.logger != nil {
		h.logger.Info("Processing Swift version check request")
	}

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
		if h.logger != nil {
			h.logger.WithError(err).Error("Failed to marshal arguments")
		}
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal arguments: %v", err)), nil
	}

	if err := json.Unmarshal(jsonData, &params); err != nil {
		if h.logger != nil {
			h.logger.WithError(err).Error("Failed to parse arguments")
		}
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
	}

	if params.Dependencies == nil {
		if h.logger != nil {
			h.logger.Error("Dependencies array is required")
		}
		return mcp.NewToolResultError("Dependencies array is required"), nil
	}

	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"dependencyCount": len(params.Dependencies),
			"hasConstraints":  params.Constraints != nil,
		}).Debug("Processing Swift version check request")
	}

	// Parse constraints
	constraints := make(map[string]*VersionConstraint)
	if params.Constraints != nil {
		for pkg, c := range params.Constraints {
			constraintData, err := json.Marshal(c)
			if err != nil {
				if h.logger != nil {
					h.logger.WithFields(logrus.Fields{
						"package": pkg,
						"error":   err.Error(),
					}).Warn("Failed to marshal constraint")
				}
				continue
			}

			var constraint VersionConstraint
			if err := json.Unmarshal(constraintData, &constraint); err != nil {
				if h.logger != nil {
					h.logger.WithFields(logrus.Fields{
						"package": pkg,
						"error":   err.Error(),
					}).Warn("Failed to parse constraint")
				}
				continue
			}

			constraints[pkg] = &constraint

			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"package":        pkg,
					"majorVersion":   constraint.MajorVersion,
					"excludePackage": constraint.ExcludePackage,
				}).Debug("Parsed constraint")
			}
		}
	}

	// Check versions for each package
	results := make([]*PackageVersion, 0, len(params.Dependencies))
	for _, dep := range params.Dependencies {
		if dep.URL == "" {
			if h.logger != nil {
				h.logger.Debug("Skipping dependency with empty URL")
			}
			continue
		}

		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"url":         dep.URL,
				"version":     dep.Version,
				"requirement": dep.Requirement,
			}).Debug("Checking Swift package version")
		}

		var constraint *VersionConstraint
		if c, ok := constraints[dep.URL]; ok {
			constraint = c
		}

		result, err := h.getPackageVersion(dep.URL, dep.Version, dep.Requirement, constraint)
		if err != nil {
			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"url":   dep.URL,
					"error": err.Error(),
				}).Error("Error checking Swift package")
			}
			fmt.Printf("Error checking Swift package %s: %v\n", dep.URL, err)
			continue
		}

		results = append(results, result)
	}

	if h.logger != nil {
		h.logger.WithField("resultCount", len(results)).Info("Completed Swift version check")
	}

	// Return results
	return NewToolResultJSON(results)
}
