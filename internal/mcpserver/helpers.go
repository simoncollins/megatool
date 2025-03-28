package mcpserver

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sirupsen/logrus"
)

// ExtractStringParam extracts a string parameter from the arguments
func ExtractStringParam(args map[string]interface{}, name string, logger *logrus.Logger) (string, bool) {
	val, ok := args[name]
	if !ok {
		if logger != nil {
			logger.WithField("parameter", name).Error("Missing required parameter")
		}
		return "", false
	}

	strVal, ok := val.(string)
	if !ok {
		if logger != nil {
			logger.WithFields(logrus.Fields{
				"parameter": name,
				"value":     val,
			}).Error("Parameter is not a string")
		}
		return "", false
	}

	return strVal, true
}

// ExtractNumberParam extracts a number parameter from the arguments
func ExtractNumberParam(args map[string]interface{}, name string, logger *logrus.Logger) (float64, bool) {
	val, ok := args[name]
	if !ok {
		if logger != nil {
			logger.WithField("parameter", name).Error("Missing required parameter")
		}
		return 0, false
	}

	numVal, ok := val.(float64)
	if !ok {
		if logger != nil {
			logger.WithFields(logrus.Fields{
				"parameter": name,
				"value":     val,
			}).Error("Parameter is not a number")
		}
		return 0, false
	}

	return numVal, true
}

// ExtractBoolParam extracts a boolean parameter from the arguments
func ExtractBoolParam(args map[string]interface{}, name string, logger *logrus.Logger) (bool, bool) {
	val, ok := args[name]
	if !ok {
		if logger != nil {
			logger.WithField("parameter", name).Error("Missing required parameter")
		}
		return false, false
	}

	boolVal, ok := val.(bool)
	if !ok {
		if logger != nil {
			logger.WithFields(logrus.Fields{
				"parameter": name,
				"value":     val,
			}).Error("Parameter is not a boolean")
		}
		return false, false
	}

	return boolVal, true
}

// ExtractArrayParam extracts an array parameter from the arguments
func ExtractArrayParam(args map[string]interface{}, name string, logger *logrus.Logger) ([]interface{}, bool) {
	val, ok := args[name]
	if !ok {
		if logger != nil {
			logger.WithField("parameter", name).Error("Missing required parameter")
		}
		return nil, false
	}

	arrayVal, ok := val.([]interface{})
	if !ok {
		if logger != nil {
			logger.WithFields(logrus.Fields{
				"parameter": name,
				"value":     val,
			}).Error("Parameter is not an array")
		}
		return nil, false
	}

	return arrayVal, true
}

// ExtractObjectParam extracts an object parameter from the arguments
func ExtractObjectParam(args map[string]interface{}, name string, logger *logrus.Logger) (map[string]interface{}, bool) {
	val, ok := args[name]
	if !ok {
		if logger != nil {
			logger.WithField("parameter", name).Error("Missing required parameter")
		}
		return nil, false
	}

	objVal, ok := val.(map[string]interface{})
	if !ok {
		if logger != nil {
			logger.WithFields(logrus.Fields{
				"parameter": name,
				"value":     val,
			}).Error("Parameter is not an object")
		}
		return nil, false
	}

	return objVal, true
}

// NewErrorResult creates a new error result
func NewErrorResult(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: message,
			},
		},
		IsError: true,
	}
}

// LogToolRequest logs a tool request
func LogToolRequest(logger *logrus.Logger, toolName string, args map[string]interface{}) {
	if logger != nil {
		logger.WithFields(logrus.Fields{
			"tool":      toolName,
			"arguments": args,
		}).Info("Tool request received")
	}
}

// HandleToolRequest is a helper function to handle tool requests
func HandleToolRequest(ctx context.Context, request mcp.CallToolRequest, handler func(ctx context.Context, args map[string]interface{}) (string, error), logger *logrus.Logger) (*mcp.CallToolResult, error) {
	// Log the request
	if logger != nil {
		logger.WithFields(logrus.Fields{
			"tool":      request.Params.Name,
			"arguments": request.Params.Arguments,
		}).Info("Tool request received")
	}

	// Call the handler
	result, err := handler(ctx, request.Params.Arguments)
	if err != nil {
		if logger != nil {
			logger.WithFields(logrus.Fields{
				"tool":  request.Params.Name,
				"error": err.Error(),
			}).Error("Tool request failed")
		}
		return NewErrorResult(fmt.Sprintf("Failed to process request: %v", err)), nil
	}

	// Return the result
	return mcp.NewToolResultText(result), nil
}
