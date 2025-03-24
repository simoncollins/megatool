package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sirupsen/logrus"
)

const (
	// GoProxyURL is the base URL for the Go proxy
	GoProxyURL = "https://proxy.golang.org"
)

// GoHandler handles Go package version checking
type GoHandler struct {
	client HTTPClient
	cache  *sync.Map
	logger *logrus.Logger
}

// NewGoHandler creates a new Go handler
func NewGoHandler(logger *logrus.Logger, cache *sync.Map) *GoHandler {
	if cache == nil {
		cache = &sync.Map{}
	}
	return &GoHandler{
		client: DefaultHTTPClient,
		cache:  cache,
		logger: logger,
	}
}

// getPackageVersions gets the available versions of a Go package
func (h *GoHandler) getPackageVersions(packagePath string) ([]string, error) {
	if h.logger != nil {
		h.logger.WithField("package", packagePath).Debug("Getting Go package versions")
	}

	// Check cache first
	if cachedVersions, ok := h.cache.Load(packagePath); ok {
		if h.logger != nil {
			h.logger.WithField("package", packagePath).Debug("Using cached Go package versions")
		}
		return cachedVersions.([]string), nil
	}

	// Make request to Go proxy
	url := fmt.Sprintf("%s/%s/@v/list", GoProxyURL, url.PathEscape(packagePath))
	if h.logger != nil {
		h.logger.WithField("url", url).Debug("Making Go proxy API request")
	}

	body, err := MakeRequestWithLogger(h.client, h.logger, "GET", url, nil)
	if err != nil {
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"package": packagePath,
				"error":   err.Error(),
			}).Error("Failed to fetch Go package")
		}
		return nil, fmt.Errorf("failed to fetch Go package %s: %w", packagePath, err)
	}

	// Parse response
	versions := strings.Split(strings.TrimSpace(string(body)), "\n")
	if len(versions) == 1 && versions[0] == "" {
		if h.logger != nil {
			h.logger.WithField("package", packagePath).Debug("No versions found")
		}
		return []string{}, nil
	}

	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"package":       packagePath,
			"versionCount":  len(versions),
			"latestVersion": versions[len(versions)-1],
		}).Debug("Found Go package versions")
	}

	// Cache the result
	h.cache.Store(packagePath, versions)

	return versions, nil
}

// getPackageVersion gets the latest version of a Go package
func (h *GoHandler) getPackageVersion(packagePath, currentVersion string) (*PackageVersion, error) {
	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"package":        packagePath,
			"currentVersion": currentVersion,
		}).Debug("Getting latest Go package version")
	}

	// Get package versions
	versions, err := h.getPackageVersions(packagePath)
	if err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		if h.logger != nil {
			h.logger.WithField("package", packagePath).Error("No versions found")
		}
		return nil, fmt.Errorf("no versions found for package %s", packagePath)
	}

	// Get latest version (last in the list)
	latestVersion := versions[len(versions)-1]

	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"package":       packagePath,
			"latestVersion": latestVersion,
		}).Debug("Found latest Go package version")
	}

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
	if h.logger != nil {
		h.logger.Info("Processing Go version check request")
	}

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

	if params.Dependencies.Module == "" {
		if h.logger != nil {
			h.logger.Error("Module name is required")
		}
		return mcp.NewToolResultError("Module name is required"), nil
	}

	if h.logger != nil {
		requireCount := 0
		if params.Dependencies.Require != nil {
			requireCount = len(params.Dependencies.Require)
		}
		replaceCount := 0
		if params.Dependencies.Replace != nil {
			replaceCount = len(params.Dependencies.Replace)
		}
		h.logger.WithFields(logrus.Fields{
			"module":       params.Dependencies.Module,
			"requireCount": requireCount,
			"replaceCount": replaceCount,
		}).Info("Checking Go package versions")
	}

	// Check versions for each package
	results := make([]*PackageVersion, 0)

	// Process required dependencies
	if params.Dependencies.Require != nil {
		if h.logger != nil {
			h.logger.WithField("count", len(params.Dependencies.Require)).Debug("Processing required dependencies")
		}
		for _, dep := range params.Dependencies.Require {
			if dep.Path == "" {
				if h.logger != nil {
					h.logger.Debug("Skipping empty dependency path")
				}
				continue
			}

			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"path":    dep.Path,
					"version": dep.Version,
				}).Debug("Checking Go package version")
			}

			result, err := h.getPackageVersion(dep.Path, dep.Version)
			if err != nil {
				if h.logger != nil {
					h.logger.WithFields(logrus.Fields{
						"path":  dep.Path,
						"error": err.Error(),
					}).Error("Error checking Go package")
				}
				fmt.Printf("Error checking Go package %s: %v\n", dep.Path, err)
				continue
			}

			results = append(results, result)
		}
	}

	// Process replaced dependencies
	if params.Dependencies.Replace != nil {
		if h.logger != nil {
			h.logger.WithField("count", len(params.Dependencies.Replace)).Debug("Processing replaced dependencies")
		}
		for _, rep := range params.Dependencies.Replace {
			if rep.Old == "" || rep.New == "" {
				if h.logger != nil {
					h.logger.WithFields(logrus.Fields{
						"old": rep.Old,
						"new": rep.New,
					}).Debug("Skipping invalid replacement")
				}
				continue
			}

			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"old":     rep.Old,
					"new":     rep.New,
					"version": rep.Version,
				}).Debug("Checking Go package version (replacement)")
			}

			result, err := h.getPackageVersion(rep.New, rep.Version)
			if err != nil {
				if h.logger != nil {
					h.logger.WithFields(logrus.Fields{
						"old":   rep.Old,
						"new":   rep.New,
						"error": err.Error(),
					}).Error("Error checking Go package")
				}
				fmt.Printf("Error checking Go package %s: %v\n", rep.New, err)
				continue
			}

			// Update name to show replacement
			result.Name = fmt.Sprintf("%s (replaces %s)", rep.New, rep.Old)

			results = append(results, result)
		}
	}

	if h.logger != nil {
		h.logger.WithField("resultCount", len(results)).Info("Completed Go version check")
	}

	// Return results
	return NewToolResultJSON(results)
}
