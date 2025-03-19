package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfigDir(t *testing.T) {
	// Test getting config directory
	serverName := "testserver"
	configDir, err := GetConfigDir(serverName)
	if err != nil {
		t.Fatalf("GetConfigDir failed: %v", err)
	}

	// Check that the path contains the server name
	if filepath.Base(filepath.Dir(configDir)) != ConfigDirName {
		t.Errorf("Expected config dir to contain %s, got %s", ConfigDirName, configDir)
	}

	// Check that the path ends with the server name
	if filepath.Base(configDir) != serverName {
		t.Errorf("Expected config dir to end with %s, got %s", serverName, configDir)
	}
}

func TestGetConfigFilePath(t *testing.T) {
	// Test getting config file path
	serverName := "testserver"
	configPath, err := GetConfigFilePath(serverName)
	if err != nil {
		t.Fatalf("GetConfigFilePath failed: %v", err)
	}

	// Check that the path contains the server name
	if filepath.Base(filepath.Dir(configPath)) != serverName {
		t.Errorf("Expected config path to contain %s, got %s", serverName, configPath)
	}

	// Check that the path ends with the default config file name
	if filepath.Base(configPath) != DefaultConfigFileName {
		t.Errorf("Expected config path to end with %s, got %s", DefaultConfigFileName, configPath)
	}
}

// mockHomeDir is a helper function to temporarily set the HOME environment variable
func mockHomeDir(t *testing.T) (string, func()) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "megatool-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Save original HOME
	origHome := os.Getenv("HOME")

	// Set HOME to temp dir
	os.Setenv("HOME", tempDir)

	// Return cleanup function
	return tempDir, func() {
		os.Setenv("HOME", origHome)
		os.RemoveAll(tempDir)
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Mock home directory
	_, cleanup := mockHomeDir(t)
	defer cleanup()

	// Test server name
	serverName := "testserver"

	// Create a test config
	testConfig := &Config{
		APIEndpoint: "https://api.example.com",
		Username:    "testuser",
	}

	// Save the config
	if err := Save(serverName, testConfig); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Check that the config file exists
	configPath, err := GetConfigFilePath(serverName)
	if err != nil {
		t.Fatalf("GetConfigFilePath failed: %v", err)
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("Config file not created at %s", configPath)
	}

	// Load the config
	loadedConfig, err := Load(serverName)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check that the loaded config matches the original
	if loadedConfig.APIEndpoint != testConfig.APIEndpoint {
		t.Errorf("Expected APIEndpoint %s, got %s", testConfig.APIEndpoint, loadedConfig.APIEndpoint)
	}
	if loadedConfig.Username != testConfig.Username {
		t.Errorf("Expected Username %s, got %s", testConfig.Username, loadedConfig.Username)
	}
}
