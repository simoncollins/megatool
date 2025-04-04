package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sirupsen/logrus"
)

const (
	// BedrockDocsURL is the URL for the AWS Bedrock documentation
	BedrockDocsURL = "https://docs.aws.amazon.com/bedrock/latest/userguide/models-supported.html"
	// BedrockCacheTTL is the time-to-live for the Bedrock models cache (1 hour)
	BedrockCacheTTL = 1 * time.Hour
)

// BedrockHandler handles AWS Bedrock model checking
type BedrockHandler struct {
	client      HTTPClient
	cache       *sync.Map
	modelsCache []BedrockModel
	lastFetch   time.Time
	cacheMutex  sync.RWMutex
	logger      *logrus.Logger
}

// NewBedrockHandler creates a new Bedrock handler
func NewBedrockHandler(logger *logrus.Logger, cache *sync.Map) *BedrockHandler {
	if cache == nil {
		cache = &sync.Map{}
	}
	return &BedrockHandler{
		client: DefaultHTTPClient,
		cache:  cache,
		logger: logger,
	}
}

// fetchModels fetches the latest Bedrock model information from AWS documentation
func (h *BedrockHandler) fetchModels() ([]BedrockModel, error) {
	if h.logger != nil {
		h.logger.Debug("Fetching Bedrock models")
	}

	// Check if we have a valid cache
	h.cacheMutex.RLock()
	if len(h.modelsCache) > 0 && time.Since(h.lastFetch) < BedrockCacheTTL {
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"modelCount": len(h.modelsCache),
				"cacheAge":   time.Since(h.lastFetch).String(),
			}).Debug("Using cached Bedrock models")
		}
		models := h.modelsCache
		h.cacheMutex.RUnlock()
		return models, nil
	}
	h.cacheMutex.RUnlock()

	// Acquire write lock
	h.cacheMutex.Lock()
	defer h.cacheMutex.Unlock()

	// Check again in case another goroutine updated the cache while we were waiting
	if len(h.modelsCache) > 0 && time.Since(h.lastFetch) < BedrockCacheTTL {
		if h.logger != nil {
			h.logger.WithFields(logrus.Fields{
				"modelCount": len(h.modelsCache),
				"cacheAge":   time.Since(h.lastFetch).String(),
			}).Debug("Using cached Bedrock models (after lock)")
		}
		return h.modelsCache, nil
	}

	if h.logger != nil {
		h.logger.WithField("url", BedrockDocsURL).Debug("Making request to AWS Bedrock documentation")
	}

	// Make request to AWS Bedrock documentation
	body, err := MakeRequestWithLogger(h.client, h.logger, "GET", BedrockDocsURL, nil)
	if err != nil {
		// If we have a cache, return it even if it's expired
		if len(h.modelsCache) > 0 {
			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"error":      err.Error(),
					"modelCount": len(h.modelsCache),
					"cacheAge":   time.Since(h.lastFetch).String(),
				}).Warn("Failed to fetch Bedrock models, using expired cache")
			}
			return h.modelsCache, nil
		}
		if h.logger != nil {
			h.logger.WithError(err).Error("Failed to fetch Bedrock models and no cache available")
		}
		return nil, fmt.Errorf("failed to fetch Bedrock models: %w", err)
	}

	// Parse the HTML to extract the model information
	if h.logger != nil {
		h.logger.Debug("Parsing Bedrock models from HTML")
	}
	models := h.parseModelsFromHTML(string(body))

	if h.logger != nil {
		h.logger.WithField("modelCount", len(models)).Info("Successfully fetched and parsed Bedrock models")
	}

	// Update cache
	h.modelsCache = models
	h.lastFetch = time.Now()

	return models, nil
}

// parseModelsFromHTML parses the HTML to extract model information from the first table
func (h *BedrockHandler) parseModelsFromHTML(html string) []BedrockModel {
	models := make([]BedrockModel, 0)

	// Find the first table in the HTML
	tableRegex := regexp.MustCompile(`<table[\s\S]*?>[\s\S]*?</table>`)
	tableMatch := tableRegex.FindString(html)
	if tableMatch == "" {
		if h.logger != nil {
			h.logger.Error("No table found in HTML")
		}
		fmt.Println("No table found in HTML")
		return models
	}

	// Extract rows from the table
	rowRegex := regexp.MustCompile(`<tr[\s\S]*?>[\s\S]*?</tr>`)
	rows := rowRegex.FindAllString(tableMatch, -1)
	if len(rows) <= 1 {
		if h.logger != nil {
			h.logger.Error("No rows found in table")
		}
		fmt.Println("No rows found in table")
		return models
	}

	if h.logger != nil {
		h.logger.WithField("rowCount", len(rows)-1).Debug("Found rows in table")
	}

	// Skip the header row
	for i := 1; i < len(rows); i++ {
		row := rows[i]

		// Extract cells from the row
		cellRegex := regexp.MustCompile(`<t[dh][\s\S]*?>[\s\S]*?</t[dh]>`)
		cells := cellRegex.FindAllString(row, -1)
		if len(cells) < 7 {
			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"rowIndex":   i,
					"cellCount":  len(cells),
					"rowPreview": row[:100] + "...",
				}).Warn("Invalid row format")
			}
			fmt.Printf("Invalid row format (cells: %d): %s\n", len(cells), row[:100]+"...")
			continue
		}

		// Extract text from cells
		provider := h.extractTextFromCell(cells[0])
		modelName := h.extractTextFromCell(cells[1])
		modelID := h.extractTextFromCell(cells[2])
		regionsSupported := h.extractRegionsFromCell(cells[3])
		inputModalities := h.extractListFromCell(cells[4])
		outputModalities := h.extractListFromCell(cells[5])
		streamingSupported := strings.ToLower(h.extractTextFromCell(cells[6])) == "yes"

		// Only add if we have valid data
		if modelName != "" && modelID != "" {
			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"provider":  provider,
					"modelName": modelName,
					"modelID":   modelID,
				}).Debug("Found Bedrock model")
			}
			models = append(models, BedrockModel{
				Provider:           provider,
				ModelName:          modelName,
				ModelID:            modelID,
				RegionsSupported:   regionsSupported,
				InputModalities:    inputModalities,
				OutputModalities:   outputModalities,
				StreamingSupported: streamingSupported,
			})
		} else {
			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"provider":  provider,
					"modelName": modelName,
					"modelID":   modelID,
				}).Warn("Skipping invalid model data")
			}
		}
	}

	return models
}

// extractTextFromCell extracts text from a table cell
func (h *BedrockHandler) extractTextFromCell(cell string) string {
	// Remove HTML tags and trim whitespace
	text := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(cell, "")
	return strings.TrimSpace(text)
}

// extractRegionsFromCell extracts a list of regions from a cell
func (h *BedrockHandler) extractRegionsFromCell(cell string) []string {
	text := h.extractTextFromCell(cell)
	// Split by whitespace and filter out empty strings
	regions := regexp.MustCompile(`\s+`).Split(text, -1)
	result := make([]string, 0, len(regions))
	for _, region := range regions {
		if region != "" {
			result = append(result, region)
		}
	}
	return result
}

// extractListFromCell extracts a list from a cell (comma-separated values)
func (h *BedrockHandler) extractListFromCell(cell string) []string {
	text := h.extractTextFromCell(cell)
	// Split by comma and trim whitespace
	items := strings.Split(text, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

// searchModels searches for Bedrock models based on query parameters
func (h *BedrockHandler) searchModels(query, provider, region string) (*BedrockModelSearchResult, error) {
	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"query":    query,
			"provider": provider,
			"region":   region,
		}).Debug("Searching Bedrock models")
	}

	models, err := h.fetchModels()
	if err != nil {
		return nil, err
	}

	// Normalize the query for case-insensitive search
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))
	normalizedProvider := strings.ToLower(strings.TrimSpace(provider))
	normalizedRegion := strings.ToLower(strings.TrimSpace(region))

	// Filter models based on query parameters
	filteredModels := make([]BedrockModel, 0)
	for _, model := range models {
		// Match query against model name or model ID (fuzzy search)
		matchesQuery := normalizedQuery == "" ||
			FuzzyMatch(strings.ToLower(model.ModelName), normalizedQuery) ||
			FuzzyMatch(strings.ToLower(model.ModelID), normalizedQuery)

		// Match provider if specified (fuzzy search)
		matchesProvider := normalizedProvider == "" ||
			FuzzyMatch(strings.ToLower(model.Provider), normalizedProvider)

		// Match region if specified
		matchesRegion := normalizedRegion == ""
		if !matchesRegion {
			for _, r := range model.RegionsSupported {
				if strings.Contains(strings.ToLower(r), normalizedRegion) ||
					strings.Contains(normalizedRegion, strings.ToLower(r)) {
					matchesRegion = true
					break
				}
			}
		}

		if matchesQuery && matchesProvider && matchesRegion {
			filteredModels = append(filteredModels, model)
		}
	}

	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"query":        query,
			"provider":     provider,
			"region":       region,
			"totalMatches": len(filteredModels),
		}).Debug("Completed Bedrock model search")
	}

	// Sort results by relevance if there's a query
	if normalizedQuery != "" {
		// Sort by exact match first, then by how early the match appears
		// This is a simplified version of the sorting in the TypeScript code
		// For a more complex sorting, we would need to implement a custom sort function
	}

	return &BedrockModelSearchResult{
		Models:     filteredModels,
		TotalCount: len(filteredModels),
	}, nil
}

// getModelByID gets a specific Bedrock model by ID
func (h *BedrockHandler) getModelByID(modelID string) (*BedrockModel, error) {
	if h.logger != nil {
		h.logger.WithField("modelID", modelID).Debug("Getting Bedrock model by ID")
	}

	models, err := h.fetchModels()
	if err != nil {
		return nil, err
	}

	for _, model := range models {
		if model.ModelID == modelID {
			if h.logger != nil {
				h.logger.WithFields(logrus.Fields{
					"modelID":   modelID,
					"modelName": model.ModelName,
					"provider":  model.Provider,
				}).Debug("Found Bedrock model by ID")
			}
			return &model, nil
		}
	}

	if h.logger != nil {
		h.logger.WithField("modelID", modelID).Error("Bedrock model not found")
	}
	return nil, fmt.Errorf("model not found: %s", modelID)
}

// getLatestClaudeSonnetModel gets the latest Claude Sonnet model
func (h *BedrockHandler) getLatestClaudeSonnetModel() (*BedrockModel, error) {
	if h.logger != nil {
		h.logger.Debug("Getting latest Claude Sonnet model")
	}

	models, err := h.fetchModels()
	if err != nil {
		return nil, err
	}

	// Filter for Claude Sonnet models
	sonnetModels := make([]BedrockModel, 0)
	for _, model := range models {
		provider := strings.ToLower(model.Provider)
		modelName := strings.ToLower(model.ModelName)
		if strings.Contains(provider, "anthropic") &&
			strings.Contains(modelName, "claude") &&
			strings.Contains(modelName, "sonnet") {
			sonnetModels = append(sonnetModels, model)
		}
	}

	if len(sonnetModels) == 0 {
		if h.logger != nil {
			h.logger.Error("No Claude Sonnet models found")
		}
		return nil, fmt.Errorf("no Claude Sonnet models found")
	}

	if h.logger != nil {
		h.logger.WithField("sonnetModelCount", len(sonnetModels)).Debug("Found Claude Sonnet models")
	}

	// Sort by version number to find the latest
	// Extract version numbers and sort
	// This is a simplified version of the sorting in the TypeScript code
	// For a more complex sorting, we would need to implement a custom sort function
	latestModel := sonnetModels[0]
	latestVersion := 0.0

	for _, model := range sonnetModels {
		// Extract version numbers (e.g., 3.5, 3.7)
		versionMatch := regexp.MustCompile(`(\d+\.\d+)`).FindStringSubmatch(model.ModelName)
		if len(versionMatch) < 2 {
			continue
		}

		version := 0.0
		_, err := fmt.Sscanf(versionMatch[1], "%f", &version)
		if err != nil {
			continue
		}

		if version > latestVersion {
			latestVersion = version
			latestModel = model
		} else if version == latestVersion {
			// If same version number, check for "v2" or similar in the name
			if strings.Contains(strings.ToLower(model.ModelName), "v2") &&
				!strings.Contains(strings.ToLower(latestModel.ModelName), "v2") {
				latestModel = model
			}
		}
	}

	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"modelName": latestModel.ModelName,
			"modelID":   latestModel.ModelID,
			"version":   latestVersion,
		}).Info("Found latest Claude Sonnet model")
	}

	return &latestModel, nil
}

// GetLatestVersion gets the latest versions for Bedrock models
func (h *BedrockHandler) GetLatestVersion(ctx context.Context, args interface{}) (*mcp.CallToolResult, error) {
	if h.logger != nil {
		h.logger.Info("Processing Bedrock model check request")
	}

	// Parse arguments
	var params struct {
		Action   string `json:"action,omitempty"`
		Query    string `json:"query,omitempty"`
		Provider string `json:"provider,omitempty"`
		Region   string `json:"region,omitempty"`
		ModelID  string `json:"modelId,omitempty"`
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

	// Set default action
	if params.Action == "" {
		params.Action = "list"
	}

	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"action":   params.Action,
			"query":    params.Query,
			"provider": params.Provider,
			"region":   params.Region,
			"modelID":  params.ModelID,
		}).Debug("Processing Bedrock request")
	}

	var result interface{}
	var fetchErr error

	switch params.Action {
	case "search":
		result, fetchErr = h.searchModels(params.Query, params.Provider, params.Region)
	case "get":
		if params.ModelID == "" {
			if h.logger != nil {
				h.logger.Error("Model ID is required for 'get' action")
			}
			return mcp.NewToolResultError("Model ID is required for 'get' action"), nil
		}
		model, err := h.getModelByID(params.ModelID)
		if err != nil {
			fetchErr = err
		} else {
			result = &BedrockModelSearchResult{
				Models:     []BedrockModel{*model},
				TotalCount: 1,
			}
		}
	case "get_latest_claude_sonnet":
		model, err := h.getLatestClaudeSonnetModel()
		if err != nil {
			if h.logger != nil {
				h.logger.WithError(err).Error("Failed to get latest Claude Sonnet model")
			}
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get latest Claude Sonnet model: %v", err)), nil
		}
		result = model
	default:
		// Default to list all models
		models, err := h.fetchModels()
		if err != nil {
			fetchErr = err
		} else {
			result = &BedrockModelSearchResult{
				Models:     models,
				TotalCount: len(models),
			}
		}
	}

	if fetchErr != nil {
		if h.logger != nil {
			h.logger.WithError(fetchErr).Error("Failed to fetch Bedrock models")
		}
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch Bedrock models: %v", fetchErr)), nil
	}

	if h.logger != nil {
		h.logger.WithField("action", params.Action).Info("Completed Bedrock model check")
	}

	// Return results
	return NewToolResultJSON(result)
}
