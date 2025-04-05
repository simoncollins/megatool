package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ClientType represents an MCP client type
type ClientType string

const (
	// ClientCline represents the VS Code Cline extension
	ClientCline ClientType = "cline"
)

// ServerConfig represents an MCP server configuration in a client's config file
type ServerConfig struct {
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Env         map[string]string `json:"env"`
	Disabled    bool              `json:"disabled"`
	AutoApprove []string          `json:"autoApprove"`
}

// ClientConfig represents the structure of an MCP client's config file
type ClientConfig struct {
	MCPServers map[string]ServerConfig `json:"mcpServers"`
}

// GetClientConfigPath returns the path to the config file for the given client
func GetClientConfigPath(clientType ClientType) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	switch clientType {
	case ClientCline:
		// Path differs based on OS
		switch runtime.GOOS {
		case "darwin":
			return filepath.Join(homeDir, "Library", "Application Support", "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"), nil
		case "linux":
			return filepath.Join(homeDir, ".config", "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"), nil
		case "windows":
			// On Windows, use %APPDATA% which typically points to C:\Users\<username>\AppData\Roaming
			return filepath.Join(os.Getenv("APPDATA"), "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"), nil
		default:
			return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
		}
	default:
		return "", fmt.Errorf("unsupported client type: %s", clientType)
	}
}

// ReadClientConfig reads the config file for the given client
func ReadClientConfig(clientType ClientType) (*ClientConfig, error) {
	configPath, err := GetClientConfigPath(clientType)
	if err != nil {
		return nil, err
	}

	// Check if the file exists
	_, err = os.Stat(configPath)
	if os.IsNotExist(err) {
		// Return empty config if file doesn't exist
		return &ClientConfig{
			MCPServers: make(map[string]ServerConfig),
		}, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to check if config file exists: %w", err)
	}

	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the JSON
	var config ClientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Initialize the map if it's nil
	if config.MCPServers == nil {
		config.MCPServers = make(map[string]ServerConfig)
	}

	return &config, nil
}

// WriteClientConfig writes the config file for the given client
func WriteClientConfig(clientType ClientType, config *ClientConfig) error {
	configPath, err := GetClientConfigPath(clientType)
	if err != nil {
		return err
	}

	// Ensure the directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal the JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write the file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// InstallServer installs a server into the given client's config
func InstallServer(clientType ClientType, serverName string) error {
	// Read the client config
	config, err := ReadClientConfig(clientType)
	if err != nil {
		return err
	}

	// Create the server config
	args := []string{"run"}

	// Add the client flag if it's not the default
	if clientType != "" {
		args = append(args, "--client", string(clientType))
	}

	// Add the server name
	args = append(args, serverName)

	serverConfig := ServerConfig{
		Command:     "megatool",
		Args:        args,
		Env:         make(map[string]string),
		Disabled:    false,
		AutoApprove: []string{},
	}

	// Add the server config to the client config
	config.MCPServers[serverName] = serverConfig

	// Write the updated config
	return WriteClientConfig(clientType, config)
}
