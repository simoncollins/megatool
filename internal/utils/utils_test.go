package utils

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestIsConfigured(t *testing.T) {
	// Mock home directory
	tempDir, cleanup := mockHomeDir(t)
	defer cleanup()

	// Test server name
	serverName := "testserver"

	// Initially, the server should not be configured
	if IsConfigured(serverName) {
		t.Errorf("Expected server %s to not be configured initially", serverName)
	}

	// Create the config directory and file
	configDir := filepath.Join(tempDir, ".config", "megatool", serverName)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configFile := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Now the server should be configured
	if !IsConfigured(serverName) {
		t.Errorf("Expected server %s to be configured after creating config file", serverName)
	}
}

func TestPrintError(t *testing.T) {
	// Redirect stderr to capture output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Call PrintError
	PrintError("Test error: %s", "something went wrong")

	// Close the writer and restore stderr
	w.Close()
	os.Stderr = oldStderr

	// Read the output
	var buf [1024]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	// Check the output
	expected := "Error: Test error: something went wrong\n"
	if output != expected {
		t.Errorf("Expected output %q, got %q", expected, output)
	}
}

func TestPrintInfo(t *testing.T) {
	// Redirect stdout to capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call PrintInfo
	PrintInfo("Test info: %s", "something happened")

	// Close the writer and restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read the output
	var buf [1024]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	// Check the output
	expected := "Test info: something happened\n"
	if output != expected {
		t.Errorf("Expected output %q, got %q", expected, output)
	}
}
