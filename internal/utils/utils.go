package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FindExecutable finds the path to an executable in the PATH
func FindExecutable(name string) (string, error) {
	return exec.LookPath(name)
}

// GetBinaryPath returns the path to a megatool binary
func GetBinaryPath(binaryName string) (string, error) {
	// First, check if the binary is in the same directory as the current executable
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	execDir := filepath.Dir(execPath)
	binaryPath := filepath.Join(execDir, binaryName)

	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath, nil
	}

	// If not found in the same directory, look in PATH
	return FindExecutable(binaryName)
}

// IsConfigured checks if a server is configured
func IsConfigured(serverName string) bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	configPath := filepath.Join(homeDir, ".config", "megatool", serverName, "config.json")
	_, err = os.Stat(configPath)
	return err == nil
}

// GetAvailableServers returns a list of available MCP server names
func GetAvailableServers() ([]string, error) {
	// Get the directory of the current executable
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	execDir := filepath.Dir(execPath)

	// Look for binaries with the pattern "megatool-*"
	entries, err := os.ReadDir(execDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var servers []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "megatool-") {
			// Extract server name from binary name
			serverName := strings.TrimPrefix(entry.Name(), "megatool-")
			servers = append(servers, serverName)
		}
	}

	return servers, nil
}

// PrintError prints an error message to stderr
func PrintError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}

// PrintInfo prints an info message to stdout
func PrintInfo(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}
