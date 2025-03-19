package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const (
	// ServiceName is the name used for keyring service
	ServiceName = "megatool"

	// ConfigDirName is the name of the config directory
	ConfigDirName = "megatool"

	// DefaultConfigFileName is the default name for config files
	DefaultConfigFileName = "config.json"
)

// Config represents the configuration for an MCP server
type Config struct {
	// Add any non-sensitive configuration fields here
	APIEndpoint string `json:"api_endpoint,omitempty"`
	Username    string `json:"username,omitempty"`
}

// GetConfigDir returns the path to the configuration directory for a server
func GetConfigDir(serverName string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", ConfigDirName, serverName)
	return configDir, nil
}

// GetConfigFilePath returns the path to the configuration file for a server
func GetConfigFilePath(serverName string) (string, error) {
	configDir, err := GetConfigDir(serverName)
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, DefaultConfigFileName), nil
}

// Load loads the configuration for a server
func Load(serverName string) (*Config, error) {
	configPath, err := GetConfigFilePath(serverName)
	if err != nil {
		return nil, err
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found, please run with --configure flag")
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse config
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// Save saves the configuration for a server
func Save(serverName string, config *Config) error {
	configDir, err := GetConfigDir(serverName)
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write config file
	configPath := filepath.Join(configDir, DefaultConfigFileName)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// StoreSecure stores a sensitive value in the system keyring
func StoreSecure(serverName, key, value string) error {
	// Use the format "megatool-{serverName}" as the service name
	service := fmt.Sprintf("%s-%s", ServiceName, serverName)

	// Store the value in the keyring
	err := keyring.Set(service, key, value)
	if err != nil {
		return fmt.Errorf("failed to store value in keyring: %w", err)
	}

	return nil
}

// GetSecure retrieves a sensitive value from the system keyring
func GetSecure(serverName, key string) (string, error) {
	// Use the format "megatool-{serverName}" as the service name
	service := fmt.Sprintf("%s-%s", ServiceName, serverName)

	// Get the value from the keyring
	value, err := keyring.Get(service, key)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve value from keyring: %w", err)
	}

	return value, nil
}

// DeleteSecure deletes a sensitive value from the system keyring
func DeleteSecure(serverName, key string) error {
	// Use the format "megatool-{serverName}" as the service name
	service := fmt.Sprintf("%s-%s", ServiceName, serverName)

	// Delete the value from the keyring
	err := keyring.Delete(service, key)
	if err != nil {
		return fmt.Errorf("failed to delete value from keyring: %w", err)
	}

	return nil
}
