package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sirupsen/logrus"
)

const (
	// MavenCentralURL is the base URL for the Maven Central repository search API
	MavenCentralURL = "https://search.maven.org/solrsearch/select"
)

// JavaHandler handles Java package version checking
type JavaHandler struct {
	client HTTPClient
	cache  *sync.Map
	logger *logrus.Logger
}

// NewJavaHandler creates a new Java handler
func NewJavaHandler(logger *logrus.Logger, cache *sync.Map) *JavaHandler {
	if cache == nil {
		cache = &sync.Map{}
	}
	return &JavaHandler{
		client: DefaultHTTPClient,
		cache:  cache,
		logger: logger,
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
	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"groupId":        groupID,
			"artifactId":     artifactID,
			"currentVersion": currentVersion,
			"scope":          scope,
		}).Debug("Getting latest Maven package version")
	}

	// Create cache key
	cacheKey := fmt.Sprintf("%s:%s", groupID, artifactID)

	// Check cache first
	if cachedInfo, ok := h.cache.Load(cacheKey); ok {
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"groupId":    groupID,
				"artifactId": artifactID,
			}).Debug("Using cached Maven package info")
		}

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
	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"url":   url,
			"query": query,
		}).Debug("Making Maven Central API request")
	}

	body, err := MakeRequestWithLogger(h.client, h.logger, "GET", url, nil)
	if err != nil {
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"groupId":    groupID,
				"artifactId": artifactID,
				"error":      err.Error(),
			}).Error("Failed to fetch Maven package")
		}
		return nil, fmt.Errorf("failed to fetch Maven package %s:%s: %w", groupID, artifactID, err)
	}

	// Parse response
	var response MavenSearchResponse
	if err := json.Unmarshal(body, &response); err != nil {
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"groupId":    groupID,
				"artifactId": artifactID,
				"error":      err.Error(),
			}).Error("Failed to parse Maven search response")
		}
		return nil, fmt.Errorf("failed to parse Maven search response: %w", err)
	}

	if response.Response.NumFound == 0 || len(response.Response.Docs) == 0 {
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"groupId":    groupID,
				"artifactId": artifactID,
			}).Error("Package not found")
		}
		return nil, fmt.Errorf("package not found: %s:%s", groupID, artifactID)
	}

	// Get latest version
	latestVersion := response.Response.Docs[0].Version

	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"groupId":       groupID,
			"artifactId":    artifactID,
			"latestVersion": latestVersion,
		}).Debug("Found latest Maven package version")
	}

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
	if h.logger != nil {
		h.logger.Info("Processing Maven version check request")
	}

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
		h.logger.WithField("dependencyCount", len(params.Dependencies)).Info("Checking Maven package versions")
	}

	// Check versions for each package
	results := make([]*PackageVersion, 0, len(params.Dependencies))
	for _, dep := range params.Dependencies {
		if dep.GroupID == "" || dep.ArtifactID == "" {
			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"groupId":    dep.GroupID,
					"artifactId": dep.ArtifactID,
				}).Debug("Skipping invalid Maven dependency")
			}
			continue
		}

		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"groupId":    dep.GroupID,
				"artifactId": dep.ArtifactID,
				"version":    dep.Version,
				"scope":      dep.Scope,
			}).Debug("Checking Maven package version")
		}

		result, err := h.getPackageVersion(dep.GroupID, dep.ArtifactID, dep.Version, dep.Scope)
		if err != nil {
			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"groupId":    dep.GroupID,
					"artifactId": dep.ArtifactID,
					"error":      err.Error(),
				}).Error("Error checking Maven package")
			}
			fmt.Printf("Error checking Maven package %s:%s: %v\n", dep.GroupID, dep.ArtifactID, err)
			continue
		}

		results = append(results, result)
	}

	if h.logger != nil {
		h.logger.WithField("resultCount", len(results)).Info("Completed Maven version check")
	}

	// Return results
	return NewToolResultJSON(results)
}

// GetLatestVersionFromGradle gets the latest versions for Gradle packages
func (h *JavaHandler) GetLatestVersionFromGradle(ctx context.Context, args interface{}) (*mcp.CallToolResult, error) {
	if h.logger != nil {
		h.logger.Info("Processing Gradle version check request")
	}

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
		h.logger.WithField("dependencyCount", len(params.Dependencies)).Info("Checking Gradle package versions")
	}

	// Check versions for each package
	results := make([]*PackageVersion, 0, len(params.Dependencies))
	for _, dep := range params.Dependencies {
		if dep.Group == "" || dep.Name == "" || dep.Configuration == "" {
			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"group":         dep.Group,
					"name":          dep.Name,
					"configuration": dep.Configuration,
				}).Debug("Skipping invalid Gradle dependency")
			}
			continue
		}

		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"group":         dep.Group,
				"name":          dep.Name,
				"version":       dep.Version,
				"configuration": dep.Configuration,
			}).Debug("Checking Gradle package version")
		}

		result, err := h.getPackageVersion(dep.Group, dep.Name, dep.Version, dep.Configuration)
		if err != nil {
			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"group": dep.Group,
					"name":  dep.Name,
					"error": err.Error(),
				}).Error("Error checking Gradle package")
			}
			fmt.Printf("Error checking Gradle package %s:%s: %v\n", dep.Group, dep.Name, err)
			continue
		}

		results = append(results, result)
	}

	if h.logger != nil {
		h.logger.WithField("resultCount", len(results)).Info("Completed Gradle version check")
	}

	// Return results
	return NewToolResultJSON(results)
}
