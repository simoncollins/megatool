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
	"github.com/sirupsen/logrus"
)

const (
	// PyPIRegistryURL is the base URL for the PyPI registry
	PyPIRegistryURL = "https://pypi.org/pypi"
)

// PythonHandler handles Python package version checking
type PythonHandler struct {
	client HTTPClient
	cache  *sync.Map
	logger *logrus.Logger
}

// NewPythonHandler creates a new Python handler
func NewPythonHandler(logger *logrus.Logger, cache *sync.Map) *PythonHandler {
	if cache == nil {
		cache = &sync.Map{}
	}
	return &PythonHandler{
		client: DefaultHTTPClient,
		cache:  cache,
		logger: logger,
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
	if h.logger != nil {
		h.logger.WithField("package", packageName).Debug("Getting PyPI package info")
	}

	// Check cache first
	if cachedInfo, ok := h.cache.Load(packageName); ok {
		if h.logger != nil {
			h.logger.WithField("package", packageName).Debug("Using cached PyPI package info")
		}
		return cachedInfo.(*PyPIPackageInfo), nil
	}

	// Make request to PyPI registry
	url := fmt.Sprintf("%s/%s/json", PyPIRegistryURL, url.PathEscape(packageName))
	body, err := MakeRequestWithLogger(h.client, h.logger, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PyPI package %s: %w", packageName, err)
	}

	// Parse response
	var info PyPIPackageInfo
	if err := json.Unmarshal(body, &info); err != nil {
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"package": packageName,
				"error":   err.Error(),
			}).Error("Failed to parse PyPI package info")
		}
		return nil, fmt.Errorf("failed to parse PyPI package info: %w", err)
	}

	// Cache the result
	h.cache.Store(packageName, &info)

	if h.logger != nil {
		h.logger.WithField("package", packageName).Debug("Successfully retrieved PyPI package info")
	}

	return &info, nil
}

// getPackageVersion gets the latest version of a PyPI package
func (h *PythonHandler) getPackageVersion(packageName, currentVersion, label string) (*PackageVersion, error) {
	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"package":        packageName,
			"currentVersion": currentVersion,
			"label":          label,
		}).Debug("Getting latest PyPI package version")
	}

	// Get package info
	info, err := h.getPackageInfo(packageName)
	if err != nil {
		return nil, err
	}

	// Get latest version
	latestVersion := info.Info.Version
	if latestVersion == "" {
		if h.logger != nil {
			h.logger.WithField("package", packageName).Error("Latest version not found")
		}
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

	if h.logger != nil {
		currentVersionStr := ""
		if currentVersion != "" {
			currentVersionStr = *result.CurrentVersion
		}
		h.logger.WithFields(logrus.Fields{
			"package":        packageName,
			"currentVersion": currentVersionStr,
			"latestVersion":  latestVersion,
			"label":          label,
		}).Debug("Got latest PyPI package version")
	}

	return result, nil
}

// parseRequirement parses a requirement string from requirements.txt
func (h *PythonHandler) parseRequirement(requirement string) (name string, version string, err error) {
	if h.logger != nil {
		h.logger.WithField("requirement", requirement).Debug("Parsing Python requirement")
	}

	// Remove comments
	if idx := strings.IndexByte(requirement, '#'); idx != -1 {
		requirement = requirement[:idx]
	}

	// Trim whitespace
	requirement = strings.TrimSpace(requirement)
	if requirement == "" {
		if h.logger != nil {
			h.logger.Debug("Empty requirement after trimming")
		}
		return "", "", fmt.Errorf("empty requirement")
	}

	// Skip options (lines starting with -)
	if strings.HasPrefix(requirement, "-") {
		if h.logger != nil {
			h.logger.WithField("requirement", requirement).Debug("Skipping option line")
		}
		return "", "", fmt.Errorf("requirement is an option")
	}

	// Parse package name and version
	re := regexp.MustCompile(`^([A-Za-z0-9_.-]+)(?:\s*([<>=!~]+.*)?)?$`)
	matches := re.FindStringSubmatch(requirement)
	if len(matches) < 2 {
		if h.logger != nil {
			h.logger.WithField("requirement", requirement).Error("Invalid requirement format")
		}
		return "", "", fmt.Errorf("invalid requirement format: %s", requirement)
	}

	name = matches[1]
	if len(matches) > 2 && matches[2] != "" {
		version = strings.TrimSpace(matches[2])
	}

	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"requirement": requirement,
			"name":        name,
			"version":     version,
		}).Debug("Successfully parsed Python requirement")
	}

	return name, version, nil
}

// GetLatestVersionFromRequirements gets the latest versions for Python packages from requirements.txt
func (h *PythonHandler) GetLatestVersionFromRequirements(ctx context.Context, args interface{}) (*mcp.CallToolResult, error) {
	if h.logger != nil {
		h.logger.Info("Processing Python requirements.txt version check request")
	}

	// Parse arguments
	var params struct {
		Requirements []string `json:"requirements"`
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

	if params.Requirements == nil {
		if h.logger != nil {
			h.logger.Error("Requirements array is required")
		}
		return mcp.NewToolResultError("Requirements array is required"), nil
	}

	if h.logger != nil {
		h.logger.WithField("requirementCount", len(params.Requirements)).Info("Checking Python package versions")
	}

	// Check versions for each package
	results := make([]*PackageVersion, 0, len(params.Requirements))
	for _, requirement := range params.Requirements {
		name, version, err := h.parseRequirement(requirement)
		if err != nil {
			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"requirement": requirement,
					"error":       err.Error(),
				}).Debug("Skipping invalid requirement")
			}
			continue
		}

		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"package": name,
				"version": version,
			}).Debug("Checking Python package version")
		}

		result, err := h.getPackageVersion(name, version, "")
		if err != nil {
			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"package": name,
					"version": version,
					"error":   err.Error(),
				}).Error("Error checking PyPI package")
			}
			fmt.Printf("Error checking PyPI package %s: %v\n", name, err)
			continue
		}

		results = append(results, result)
	}

	if h.logger != nil {
		h.logger.WithField("resultCount", len(results)).Info("Completed Python requirements.txt version check")
	}

	// Return results
	return NewToolResultJSON(results)
}

// GetLatestVersionFromPyProject gets the latest versions for Python packages from pyproject.toml
func (h *PythonHandler) GetLatestVersionFromPyProject(ctx context.Context, args interface{}) (*mcp.CallToolResult, error) {
	if h.logger != nil {
		h.logger.Info("Processing Python pyproject.toml version check request")
	}

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

	if params.Dependencies.Dependencies == nil &&
		params.Dependencies.OptionalDependencies == nil &&
		params.Dependencies.DevDependencies == nil {
		if h.logger != nil {
			h.logger.Error("No dependencies found in pyproject.toml")
		}
		return mcp.NewToolResultError("No dependencies found in pyproject.toml"), nil
	}

	if h.logger != nil {
		dependencyCount := 0
		if params.Dependencies.Dependencies != nil {
			dependencyCount += len(params.Dependencies.Dependencies)
		}
		if params.Dependencies.OptionalDependencies != nil {
			for _, deps := range params.Dependencies.OptionalDependencies {
				dependencyCount += len(deps)
			}
		}
		if params.Dependencies.DevDependencies != nil {
			dependencyCount += len(params.Dependencies.DevDependencies)
		}
		h.logger.WithField("dependencyCount", dependencyCount).Info("Checking Python package versions")
	}

	// Check versions for each package
	results := make([]*PackageVersion, 0)

	// Process main dependencies
	if params.Dependencies.Dependencies != nil {
		if h.logger != nil {
			h.logger.WithField("count", len(params.Dependencies.Dependencies)).Debug("Processing main dependencies")
		}
		for name, version := range params.Dependencies.Dependencies {
			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"package": name,
					"version": version,
				}).Debug("Checking Python package version")
			}
			result, err := h.getPackageVersion(name, version, "")
			if err != nil {
				if h.logger != nil {
					h.logger.WithFields(logrus.Fields{
						"package": name,
						"version": version,
						"error":   err.Error(),
					}).Error("Error checking PyPI package")
				}
				fmt.Printf("Error checking PyPI package %s: %v\n", name, err)
				continue
			}
			results = append(results, result)
		}
	}

	// Process optional dependencies
	if params.Dependencies.OptionalDependencies != nil {
		if h.logger != nil {
			h.logger.WithField("groupCount", len(params.Dependencies.OptionalDependencies)).Debug("Processing optional dependencies")
		}
		for group, deps := range params.Dependencies.OptionalDependencies {
			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"group": group,
					"count": len(deps),
				}).Debug("Processing optional dependency group")
			}
			for name, version := range deps {
				if h.logger != nil {
					h.logger.WithFields(logrus.Fields{
						"package": name,
						"version": version,
						"group":   group,
					}).Debug("Checking Python package version")
				}
				result, err := h.getPackageVersion(name, version, fmt.Sprintf("optional: %s", group))
				if err != nil {
					if h.logger != nil {
						h.logger.WithFields(logrus.Fields{
							"package": name,
							"version": version,
							"group":   group,
							"error":   err.Error(),
						}).Error("Error checking PyPI package")
					}
					fmt.Printf("Error checking PyPI package %s: %v\n", name, err)
					continue
				}
				results = append(results, result)
			}
		}
	}

	// Process dev dependencies
	if params.Dependencies.DevDependencies != nil {
		if h.logger != nil {
			h.logger.WithField("count", len(params.Dependencies.DevDependencies)).Debug("Processing dev dependencies")
		}
		for name, version := range params.Dependencies.DevDependencies {
			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"package": name,
					"version": version,
				}).Debug("Checking Python package version")
			}
			result, err := h.getPackageVersion(name, version, "dev")
			if err != nil {
				if h.logger != nil {
					h.logger.WithFields(logrus.Fields{
						"package": name,
						"version": version,
						"error":   err.Error(),
					}).Error("Error checking PyPI package")
				}
				fmt.Printf("Error checking PyPI package %s: %v\n", name, err)
				continue
			}
			results = append(results, result)
		}
	}

	if h.logger != nil {
		h.logger.WithField("resultCount", len(results)).Info("Completed Python pyproject.toml version check")
	}

	// Return results
	return NewToolResultJSON(results)
}
