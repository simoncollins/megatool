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
	"github.com/sirupsen/logrus"
)

const (
	// NpmRegistryURL is the base URL for the npm registry
	NpmRegistryURL = "https://registry.npmjs.org"
)

// NpmHandler handles npm package version checking
type NpmHandler struct {
	client HTTPClient
	cache  *sync.Map
	logger *logrus.Logger
}

// NewNpmHandler creates a new npm handler
func NewNpmHandler(logger *logrus.Logger, cache *sync.Map) *NpmHandler {
	if cache == nil {
		cache = &sync.Map{}
	}
	return &NpmHandler{
		client: DefaultHTTPClient,
		cache:  cache,
		logger: logger,
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
	if h.logger != nil {
		h.logger.WithField("package", packageName).Debug("Getting npm package info")
	}

	// Check cache first
	if cachedInfo, ok := h.cache.Load(packageName); ok {
		if h.logger != nil {
			h.logger.WithField("package", packageName).Debug("Using cached npm package info")
		}
		return cachedInfo.(*NpmPackageInfo), nil
	}

	// Make request to npm registry
	url := fmt.Sprintf("%s/%s", NpmRegistryURL, url.PathEscape(packageName))
	body, err := MakeRequestWithLogger(h.client, h.logger, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch npm package %s: %w", packageName, err)
	}

	// Parse response
	var info NpmPackageInfo
	if err := json.Unmarshal(body, &info); err != nil {
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"package": packageName,
				"error":   err.Error(),
			}).Error("Failed to parse npm package info")
		}
		return nil, fmt.Errorf("failed to parse npm package info: %w", err)
	}

	// Cache the result
	h.cache.Store(packageName, &info)

	if h.logger != nil {
		h.logger.WithField("package", packageName).Debug("Successfully retrieved npm package info")
	}

	return &info, nil
}

// getPackageVersion gets the latest version of an npm package
func (h *NpmHandler) getPackageVersion(packageName, currentVersion string, constraint *VersionConstraint) (*PackageVersion, error) {
	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"package":        packageName,
			"currentVersion": currentVersion,
		}).Debug("Getting latest npm package version")
	}
	// Check if package should be excluded
	if constraint != nil && constraint.ExcludePackage {
		cleanVersion := CleanVersion(currentVersion)
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"package":        packageName,
				"currentVersion": cleanVersion,
			}).Info("Package excluded from updates")
		}
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
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"package":      packageName,
				"majorVersion": targetMajor,
			}).Debug("Limiting to specific major version")
		}
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
			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"package":       packageName,
					"majorVersion":  targetMajor,
					"latestVersion": latestVersion,
				}).Debug("Found latest version for major version")
			}
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
	if h.logger != nil {
		h.logger.Info("Processing npm version check request")
	}
	// Parse arguments
	var params struct {
		Dependencies map[string]string      `json:"dependencies"`
		Constraints  map[string]interface{} `json:"constraints"`
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
			h.logger.Error("Dependencies object is required")
		}
		return mcp.NewToolResultError("Dependencies object is required"), nil
	}

	if h.logger != nil {
		h.logger.WithField("dependencyCount", len(params.Dependencies)).Info("Checking npm package versions")
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
			if h.logger != nil {
				h.logger.WithField("package", name).Debug("Skipping package with empty version")
			}
			continue
		}

		var constraint *VersionConstraint
		if c, ok := constraints[name]; ok {
			constraint = c
		}

		result, err := h.getPackageVersion(name, version, constraint)
		if err != nil {
			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"package": name,
					"version": version,
					"error":   err.Error(),
				}).Error("Error checking npm package")
			}
			fmt.Printf("Error checking npm package %s: %v\n", name, err)
			continue
		}

		results = append(results, result)
	}

	if h.logger != nil {
		h.logger.WithField("resultCount", len(results)).Info("Completed npm version check")
	}

	// Return results
	return NewToolResultJSON(results)
}
