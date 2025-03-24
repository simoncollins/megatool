package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sirupsen/logrus"
)

const (
	// DockerHubAuthURL is the URL for Docker Hub authentication
	DockerHubAuthURL = "https://auth.docker.io/token"
	// DockerHubRegistryURL is the URL for Docker Hub registry API
	DockerHubRegistryURL = "https://registry.hub.docker.com/v2"
	// DockerHubAPIURL is the URL for Docker Hub API
	DockerHubAPIURL = "https://hub.docker.com/v2"
	// GHCRRegistryURL is the URL for GitHub Container Registry
	GHCRRegistryURL = "https://ghcr.io/v2"
)

// DockerHandler handles Docker image tag checking
type DockerHandler struct {
	client HTTPClient
	cache  *sync.Map
	logger *logrus.Logger
}

// NewDockerHandler creates a new Docker handler
func NewDockerHandler(logger *logrus.Logger, cache *sync.Map) *DockerHandler {
	if cache == nil {
		cache = &sync.Map{}
	}
	return &DockerHandler{
		client: DefaultHTTPClient,
		cache:  cache,
		logger: logger,
	}
}

// DockerTagsResponse represents a response from the Docker registry API
type DockerTagsResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

// DockerTagInfo represents information about a Docker tag
type DockerTagInfo struct {
	LastUpdated string `json:"last_updated"`
	FullSize    int64  `json:"full_size"`
}

// getDockerHubToken gets an authentication token for Docker Hub
func (h *DockerHandler) getDockerHubToken(repository string) (string, error) {
	if h.logger != nil {
		h.logger.WithField("repository", repository).Debug("Getting Docker Hub token")
	}

	// Check cache first
	cacheKey := fmt.Sprintf("dockerhub-token:%s", repository)
	if cachedToken, ok := h.cache.Load(cacheKey); ok {
		if h.logger != nil {
			h.logger.WithField("repository", repository).Debug("Using cached Docker Hub token")
		}
		return cachedToken.(string), nil
	}

	// Format repository for Docker Hub (add library/ prefix for official images)
	if !strings.Contains(repository, "/") {
		repository = "library/" + repository
	}

	// Build URL
	params := url.Values{}
	params.Set("service", "registry.docker.io")
	params.Set("scope", fmt.Sprintf("repository:%s:pull", repository))
	tokenURL := fmt.Sprintf("%s?%s", DockerHubAuthURL, params.Encode())

	if h.logger != nil {
		h.logger.WithField("url", tokenURL).Debug("Making Docker Hub token request")
	}

	// Make request
	body, err := MakeRequestWithLogger(h.client, h.logger, "GET", tokenURL, nil)
	if err != nil {
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"repository": repository,
				"error":      err.Error(),
			}).Error("Failed to get Docker Hub token")
		}
		return "", fmt.Errorf("failed to get Docker Hub token: %w", err)
	}

	// Parse response
	var response struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"repository": repository,
				"error":      err.Error(),
			}).Error("Failed to parse Docker Hub token response")
		}
		return "", fmt.Errorf("failed to parse Docker Hub token response: %w", err)
	}

	if response.Token == "" {
		if h.logger != nil {
			h.logger.WithField("repository", repository).Error("Empty token received from Docker Hub")
		}
		return "", fmt.Errorf("empty token received from Docker Hub")
	}

	// Cache the token
	h.cache.Store(cacheKey, response.Token)

	if h.logger != nil {
		h.logger.WithField("repository", repository).Debug("Successfully got Docker Hub token")
	}

	return response.Token, nil
}

// getGHCRToken gets an authentication token for GitHub Container Registry
func (h *DockerHandler) getGHCRToken() string {
	if h.logger != nil {
		h.logger.Debug("Getting GitHub Container Registry token")
	}

	// GitHub Container Registry can be accessed anonymously for public images
	// For private images, a token would be needed from environment variables
	token := os.Getenv("GITHUB_TOKEN")

	if h.logger != nil {
		if token != "" {
			h.logger.Debug("Using GitHub token from environment")
		} else {
			h.logger.Debug("No GitHub token found, using anonymous access")
		}
	}

	return token
}

// getCustomRegistryAuth gets authentication for a custom registry
func (h *DockerHandler) getCustomRegistryAuth() string {
	if h.logger != nil {
		h.logger.Debug("Getting custom registry authentication")
	}

	// For custom registries, authentication would typically be provided via environment variables
	username := os.Getenv("CUSTOM_REGISTRY_USERNAME")
	password := os.Getenv("CUSTOM_REGISTRY_PASSWORD")
	token := os.Getenv("CUSTOM_REGISTRY_TOKEN")

	if token != "" {
		if h.logger != nil {
			h.logger.Debug("Using token authentication for custom registry")
		}
		return "Bearer " + token
	} else if username != "" && password != "" {
		if h.logger != nil {
			h.logger.Debug("Using basic authentication for custom registry")
		}
		auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
		return "Basic " + auth
	}

	if h.logger != nil {
		h.logger.Debug("No authentication found for custom registry")
	}

	return ""
}

// getTags gets the tags for a Docker image
func (h *DockerHandler) getTags(registryURL, repository, authHeader string, limit int) ([]string, map[string]string, error) {
	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"registry":   registryURL,
			"repository": repository,
			"limit":      limit,
		}).Debug("Getting Docker image tags")
	}

	// Check cache first
	cacheKey := fmt.Sprintf("docker-tags:%s:%s", registryURL, repository)
	if cachedInfo, ok := h.cache.Load(cacheKey); ok {
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"registry":   registryURL,
				"repository": repository,
			}).Debug("Using cached Docker image tags")
		}
		info := cachedInfo.(map[string]interface{})
		return info["tags"].([]string), info["digests"].(map[string]string), nil
	}

	// Build URL
	tagsURL := fmt.Sprintf("%s/%s/tags/list", registryURL, repository)

	// Set headers
	headers := make(map[string]string)
	if authHeader != "" {
		headers["Authorization"] = authHeader
	}

	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"url":      tagsURL,
			"hasAuth":  authHeader != "",
		}).Debug("Making Docker tags request")
	}

	// Make request
	body, err := MakeRequestWithLogger(h.client, h.logger, "GET", tagsURL, headers)
	if err != nil {
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"registry":   registryURL,
				"repository": repository,
				"error":      err.Error(),
			}).Error("Failed to get Docker tags")
		}
		return nil, nil, fmt.Errorf("failed to get Docker tags: %w", err)
	}

	// Parse response
	var response DockerTagsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"registry":   registryURL,
				"repository": repository,
				"error":      err.Error(),
			}).Error("Failed to parse Docker tags response")
		}
		return nil, nil, fmt.Errorf("failed to parse Docker tags response: %w", err)
	}

	// Limit the number of tags
	tags := response.Tags
	if limit > 0 && len(tags) > limit {
		tags = tags[:limit]
	}

	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"registry":   registryURL,
			"repository": repository,
			"tagCount":   len(tags),
		}).Debug("Successfully got Docker image tags")
	}

	// Get digest for each tag
	digests := make(map[string]string)
	for _, tag := range tags {
		manifestURL := fmt.Sprintf("%s/%s/manifests/%s", registryURL, repository, tag)
		manifestHeaders := make(map[string]string)
		if authHeader != "" {
			manifestHeaders["Authorization"] = authHeader
		}
		manifestHeaders["Accept"] = "application/vnd.docker.distribution.manifest.v2+json"

		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"registry":   registryURL,
				"repository": repository,
				"tag":        tag,
				"url":        manifestURL,
			}).Debug("Getting Docker image manifest")
		}

		// Make request
		_, err := MakeRequestWithLogger(h.client, h.logger, "HEAD", manifestURL, manifestHeaders)
		if err != nil {
			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"registry":   registryURL,
					"repository": repository,
					"tag":        tag,
					"error":      err.Error(),
				}).Warn("Failed to get Docker image manifest")
			}
			continue
		}

		// TODO: Extract digest from response headers
		// This would require modifying the MakeRequest function to return headers
		// For now, we'll leave digests empty
	}

	// Cache the result
	h.cache.Store(cacheKey, map[string]interface{}{
		"tags":    tags,
		"digests": digests,
	})

	return tags, digests, nil
}

// getDockerHubTagInfo gets additional information about a Docker Hub tag
func (h *DockerHandler) getDockerHubTagInfo(repository, tag string) (*DockerTagInfo, error) {
	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"repository": repository,
			"tag":        tag,
		}).Debug("Getting Docker Hub tag info")
	}

	// Format repository for Docker Hub API
	if !strings.Contains(repository, "/") {
		repository = "library/" + repository
	}

	// Build URL
	tagURL := fmt.Sprintf("%s/repositories/%s/tags/%s", DockerHubAPIURL, repository, tag)

	if h.logger != nil {
		h.logger.WithField("url", tagURL).Debug("Making Docker Hub tag info request")
	}

	// Make request
	body, err := MakeRequestWithLogger(h.client, h.logger, "GET", tagURL, nil)
	if err != nil {
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"repository": repository,
				"tag":        tag,
				"error":      err.Error(),
			}).Warn("Failed to get Docker Hub tag info")
		}
		return nil, fmt.Errorf("failed to get Docker Hub tag info: %w", err)
	}

	// Parse response
	var response struct {
		LastUpdated string `json:"last_updated"`
		FullSize    int64  `json:"full_size"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"repository": repository,
				"tag":        tag,
				"error":      err.Error(),
			}).Error("Failed to parse Docker Hub tag info response")
		}
		return nil, fmt.Errorf("failed to parse Docker Hub tag info response: %w", err)
	}

	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"repository":  repository,
			"tag":         tag,
			"lastUpdated": response.LastUpdated,
			"size":        response.FullSize,
		}).Debug("Successfully got Docker Hub tag info")
	}

	return &DockerTagInfo{
		LastUpdated: response.LastUpdated,
		FullSize:    response.FullSize,
	}, nil
}

// getDockerHubTags gets the tags for a Docker Hub image
func (h *DockerHandler) getDockerHubTags(image string, limit int, includeDigest bool) ([]*DockerImageVersion, error) {
	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"image":         image,
			"limit":         limit,
			"includeDigest": includeDigest,
		}).Debug("Getting Docker Hub tags")
	}

	// Format repository for Docker Hub
	repository := image
	if !strings.Contains(repository, "/") {
		repository = "library/" + repository
	}

	// Get token
	token, err := h.getDockerHubToken(repository)
	if err != nil {
		return nil, err
	}

	// Get tags
	tags, digests, err := h.getTags(DockerHubRegistryURL, repository, "Bearer "+token, limit)
	if err != nil {
		return nil, err
	}

	// Create results
	results := make([]*DockerImageVersion, 0, len(tags))
	for _, tag := range tags {
		result := &DockerImageVersion{
			Name:     image,
			Tag:      tag,
			Registry: "dockerhub",
		}

		if includeDigest {
			if digest, ok := digests[tag]; ok {
				result.Digest = StringPtr(digest)
			}
		}

		// Try to get additional info
		info, err := h.getDockerHubTagInfo(repository, tag)
		if err == nil && info != nil {
			result.Created = StringPtr(info.LastUpdated)
			if info.FullSize > 0 {
				size := fmt.Sprintf("%d MB", info.FullSize/(1024*1024))
				result.Size = StringPtr(size)
			}
		}

		results = append(results, result)
	}

	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"image":      image,
			"resultCount": len(results),
		}).Debug("Successfully got Docker Hub tags")
	}

	return results, nil
}

// getGHCRTags gets the tags for a GitHub Container Registry image
func (h *DockerHandler) getGHCRTags(image string, limit int, includeDigest bool) ([]*DockerImageVersion, error) {
	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"image":         image,
			"limit":         limit,
			"includeDigest": includeDigest,
		}).Debug("Getting GitHub Container Registry tags")
	}

	// Get token
	token := h.getGHCRToken()
	var authHeader string
	if token != "" {
		authHeader = "Bearer " + token
	}

	// Get tags
	tags, digests, err := h.getTags(GHCRRegistryURL, image, authHeader, limit)
	if err != nil {
		return nil, err
	}

	// Create results
	results := make([]*DockerImageVersion, 0, len(tags))
	for _, tag := range tags {
		result := &DockerImageVersion{
			Name:     image,
			Tag:      tag,
			Registry: "ghcr",
		}

		if includeDigest {
			if digest, ok := digests[tag]; ok {
				result.Digest = StringPtr(digest)
			}
		}

		results = append(results, result)
	}

	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"image":      image,
			"resultCount": len(results),
		}).Debug("Successfully got GitHub Container Registry tags")
	}

	return results, nil
}

// getCustomRegistryTags gets the tags for a custom registry image
func (h *DockerHandler) getCustomRegistryTags(registry, image string, limit int, includeDigest bool) ([]*DockerImageVersion, error) {
	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"registry":      registry,
			"image":         image,
			"limit":         limit,
			"includeDigest": includeDigest,
		}).Debug("Getting custom registry tags")
	}

	// Get authentication
	authHeader := h.getCustomRegistryAuth()

	// Remove protocol and trailing slash from registry URL
	registry = strings.TrimPrefix(registry, "http://")
	registry = strings.TrimPrefix(registry, "https://")
	registry = strings.TrimSuffix(registry, "/")

	// Build registry URL
	registryURL := fmt.Sprintf("https://%s/v2", registry)

	// Get tags
	tags, digests, err := h.getTags(registryURL, image, authHeader, limit)
	if err != nil {
		return nil, err
	}

	// Create results
	results := make([]*DockerImageVersion, 0, len(tags))
	for _, tag := range tags {
		result := &DockerImageVersion{
			Name:     image,
			Tag:      tag,
			Registry: "custom",
		}

		if includeDigest {
			if digest, ok := digests[tag]; ok {
				result.Digest = StringPtr(digest)
			}
		}

		results = append(results, result)
	}

	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"registry":   registry,
			"image":      image,
			"resultCount": len(results),
		}).Debug("Successfully got custom registry tags")
	}

	return results, nil
}

// GetLatestVersion gets the latest tags for Docker images
func (h *DockerHandler) GetLatestVersion(ctx context.Context, args interface{}) (*mcp.CallToolResult, error) {
	if h.logger != nil {
		h.logger.Info("Processing Docker tags check request")
	}

	// Parse arguments
	var params struct {
		Image          string   `json:"image"`
		Registry       string   `json:"registry,omitempty"`
		CustomRegistry string   `json:"customRegistry,omitempty"`
		Limit          int      `json:"limit,omitempty"`
		FilterTags     []string `json:"filterTags,omitempty"`
		IncludeDigest  bool     `json:"includeDigest,omitempty"`
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

	if params.Image == "" {
		if h.logger != nil {
			h.logger.Error("Image name is required")
		}
		return mcp.NewToolResultError("Image name is required"), nil
	}

	// Set default values
	if params.Limit <= 0 {
		params.Limit = 10
	}
	if params.Registry == "" {
		params.Registry = "dockerhub"
	}

	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"image":          params.Image,
			"registry":       params.Registry,
			"customRegistry": params.CustomRegistry,
			"limit":          params.Limit,
			"filterTagCount": len(params.FilterTags),
			"includeDigest":  params.IncludeDigest,
		}).Debug("Processing Docker tags request")
	}

	// Get tags based on registry
	var results []*DockerImageVersion
	var fetchErr error

	switch params.Registry {
	case "ghcr":
		results, fetchErr = h.getGHCRTags(params.Image, params.Limit, params.IncludeDigest)
	case "custom":
		if params.CustomRegistry == "" {
			if h.logger != nil {
				h.logger.Error("Custom registry URL is required when registry type is \"custom\"")
			}
			return mcp.NewToolResultError("Custom registry URL is required when registry type is \"custom\""), nil
		}
		results, fetchErr = h.getCustomRegistryTags(params.CustomRegistry, params.Image, params.Limit, params.IncludeDigest)
	case "dockerhub":
		fallthrough
	default:
		results, fetchErr = h.getDockerHubTags(params.Image, params.Limit, params.IncludeDigest)
	}

	if fetchErr != nil {
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"image":   params.Image,
				"registry": params.Registry,
				"error":    fetchErr.Error(),
			}).Error("Failed to fetch Docker image tags")
		}
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch Docker image tags: %v", fetchErr)), nil
	}

	// Filter tags if filterTags is provided
	if len(params.FilterTags) > 0 {
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"image":         params.Image,
				"filterTagCount": len(params.FilterTags),
				"resultCount":    len(results),
			}).Debug("Filtering Docker tags")
		}

		filteredResults := make([]*DockerImageVersion, 0)
		for _, result := range results {
			for _, pattern := range params.FilterTags {
				matched, err := regexp.MatchString(pattern, result.Tag)
				if err == nil && matched {
					filteredResults = append(filteredResults, result)
					break
				}
			}
		}
		results = filteredResults

		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"image":              params.Image,
				"filteredResultCount": len(results),
			}).Debug("Filtered Docker tags")
		}
	}

	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"image":       params.Image,
			"registry":    params.Registry,
			"resultCount": len(results),
		}).Info("Completed Docker tags check")
	}

	// Return results
	return NewToolResultJSON(results)
}
