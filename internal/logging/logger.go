package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
)

const (
	// MaxLogFiles is the maximum number of log files to keep
	MaxLogFiles = 5
	// MaxLogSize is the maximum size of each log file in megabytes
	MaxLogSize = 10
	// MaxLogAge is the maximum age of log files in days
	MaxLogAge = 30
)

// Logger is a wrapper around logrus.Logger
type Logger struct {
	*logrus.Logger
	ServerName string
	PID        int
	FilePath   string
}

// GetLogDirectory returns the path to the log directory
func GetLogDirectory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	logDir := filepath.Join(homeDir, ".megatool", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory: %w", err)
	}

	return logDir, nil
}

// GetServerLogDirectory returns the path to a server's log directory
func GetServerLogDirectory(serverName string) (string, error) {
	baseDir, err := GetLogDirectory()
	if err != nil {
		return "", err
	}

	serverDir := filepath.Join(baseDir, serverName)
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create server log directory: %w", err)
	}

	return serverDir, nil
}

// GetLogFilePath returns the path to a log file for a server and PID
func GetLogFilePath(serverName string, pid int) (string, error) {
	serverDir, err := GetServerLogDirectory(serverName)
	if err != nil {
		return "", err
	}

	return filepath.Join(serverDir, fmt.Sprintf("server_%d.log", pid)), nil
}

// NewLogger creates a new logger for an MCP server
func NewLogger(serverName string, pid int) (*Logger, error) {
	// Create a new logrus logger
	log := logrus.New()

	// Set JSON formatter for structured logging
	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	// Get log file path
	logFilePath, err := GetLogFilePath(serverName, pid)
	if err != nil {
		return nil, fmt.Errorf("failed to get log file path: %w", err)
	}

	// Set up log rotation with lumberjack
	logFile := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    MaxLogSize,
		MaxBackups: MaxLogFiles,
		MaxAge:     MaxLogAge,
		Compress:   true,
	}

	// Create a multi-writer to write to both file and stderr
	multiWriter := io.MultiWriter(logFile, os.Stderr)
	log.SetOutput(multiWriter)

	// Set default fields
	log.WithFields(logrus.Fields{
		"server": serverName,
		"pid":    pid,
	})

	return &Logger{
		Logger:     log,
		ServerName: serverName,
		PID:        pid,
		FilePath:   logFilePath,
	}, nil
}

// WithField adds a field to the logger
func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	return l.Logger.WithFields(logrus.Fields{
		key:      value,
		"server": l.ServerName,
		"pid":    l.PID,
	})
}

// WithFields adds multiple fields to the logger
func (l *Logger) WithFields(fields logrus.Fields) *logrus.Entry {
	// Add server and PID fields
	fields["server"] = l.ServerName
	fields["pid"] = l.PID

	return l.Logger.WithFields(fields)
}

// GetLogWriter returns an io.Writer that writes to the log file
func (l *Logger) GetLogWriter() io.Writer {
	return l.Logger.Writer()
}
