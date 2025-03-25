package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ServerRecord represents a running MCP server
type ServerRecord struct {
	Name      string    `json:"name"`
	PID       int       `json:"pid"`
	StartTime time.Time `json:"start_time"`
	Client    string    `json:"client,omitempty"`
}

// ServerRecordOptions contains optional fields for server records
type ServerRecordOptions struct {
	Client string
}

// GetServerRecordsPath returns the path to the server records file
func GetServerRecordsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".config", "megatool")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "running-servers.json"), nil
}

// ReadServerRecords reads the server records from disk
func ReadServerRecords() ([]ServerRecord, error) {
	path, err := GetServerRecordsPath()
	if err != nil {
		return nil, err
	}

	// If file doesn't exist yet, return empty records
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []ServerRecord{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var records struct {
		Servers []ServerRecord `json:"servers"`
	}

	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}

	return records.Servers, nil
}

// WriteServerRecords writes the server records to disk
func WriteServerRecords(records []ServerRecord) error {
	path, err := GetServerRecordsPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(struct {
		Servers []ServerRecord `json:"servers"`
	}{Servers: records}, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// AddServerRecord adds a server record to the list
func AddServerRecord(name string, pid int, opts ...ServerRecordOptions) error {
	records, err := ReadServerRecords()
	if err != nil {
		return err
	}

	// Clean up stale records first
	records = CleanupStaleRecords(records)

	// Create new record
	record := ServerRecord{
		Name:      name,
		PID:       pid,
		StartTime: time.Now(),
	}

	// Apply options if provided
	if len(opts) > 0 {
		record.Client = opts[0].Client
	}

	// Add new record
	records = append(records, record)

	return WriteServerRecords(records)
}

// CleanupStaleRecords removes records of processes that are no longer running
func CleanupStaleRecords(records []ServerRecord) []ServerRecord {
	var active []ServerRecord

	for _, record := range records {
		if IsProcessRunning(record.PID) {
			active = append(active, record)
		}
	}

	return active
}

// FormatUptime formats the uptime of a server in a human-readable format
func FormatUptime(startTime time.Time) string {
	duration := time.Since(startTime)

	// Round to seconds
	duration = duration.Round(time.Second)

	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	} else {
		return fmt.Sprintf("%ds", seconds)
	}
}
