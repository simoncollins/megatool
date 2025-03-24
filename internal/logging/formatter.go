package logging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

// ColoredFormatter is a custom logrus formatter that adds color to log output
type ColoredFormatter struct {
	// TimestampFormat is the format used for timestamps
	TimestampFormat string
	// Colors maps server names to color functions
	Colors map[string]func(format string, a ...interface{}) string
	// DefaultColor is the color function used for servers without a specific color
	DefaultColor func(format string, a ...interface{}) string
	// ColorMutex protects the Colors map
	ColorMutex sync.RWMutex
	// ServerColors is a map of server names to color functions
	ServerColors map[string]func(format string, a ...interface{}) string
}

// Available colors for servers
var availableColors = []func(format string, a ...interface{}) string{
	color.New(color.FgCyan).SprintfFunc(),
	color.New(color.FgGreen).SprintfFunc(),
	color.New(color.FgYellow).SprintfFunc(),
	color.New(color.FgBlue).SprintfFunc(),
	color.New(color.FgMagenta).SprintfFunc(),
	color.New(color.FgRed).SprintfFunc(),
	color.New(color.FgHiCyan).SprintfFunc(),
	color.New(color.FgHiGreen).SprintfFunc(),
	color.New(color.FgHiYellow).SprintfFunc(),
	color.New(color.FgHiBlue).SprintfFunc(),
	color.New(color.FgHiMagenta).SprintfFunc(),
}

// NewColoredFormatter creates a new ColoredFormatter
func NewColoredFormatter() *ColoredFormatter {
	return &ColoredFormatter{
		TimestampFormat: "2006-01-02 15:04:05.000",
		Colors: map[string]func(format string, a ...interface{}) string{
			"calculator":      color.New(color.FgCyan).SprintfFunc(),
			"github":          color.New(color.FgGreen).SprintfFunc(),
			"package-version": color.New(color.FgYellow).SprintfFunc(),
		},
		DefaultColor: color.New(color.FgWhite).SprintfFunc(),
		ServerColors: make(map[string]func(format string, a ...interface{}) string),
	}
}

// Format formats a logrus entry
func (f *ColoredFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	// Format timestamp
	timestamp := entry.Time.Format(f.TimestampFormat)
	b.WriteString(timestamp)
	b.WriteString(" ")

	// Format log level
	levelColor := f.getLevelColor(entry.Level)
	b.WriteString(levelColor(strings.ToUpper(entry.Level.String())))
	b.WriteString(" ")

	// Format server name if present
	if server, ok := entry.Data["server"]; ok {
		serverName := fmt.Sprintf("%v", server)
		serverColor := f.getServerColor(serverName)
		b.WriteString("[")
		b.WriteString(serverColor(serverName))
		if pid, ok := entry.Data["pid"]; ok {
			b.WriteString(fmt.Sprintf(":%v", pid))
		}
		b.WriteString("] ")
		
		// Remove server and pid from fields to avoid duplication
		delete(entry.Data, "server")
		delete(entry.Data, "pid")
	}

	// Format message
	b.WriteString(entry.Message)

	// Format fields
	if len(entry.Data) > 0 {
		b.WriteString(" ")
		f.writeFields(b, entry)
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}

// writeFields writes the fields to the buffer
func (f *ColoredFormatter) writeFields(b *bytes.Buffer, entry *logrus.Entry) {
	// Get keys and sort them
	keys := make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, key := range keys {
		if i > 0 {
			b.WriteString(" ")
		}

		// Write key=value
		fmt.Fprintf(b, "%s=%v", key, entry.Data[key])
	}
}

// getLevelColor returns the color function for a log level
func (f *ColoredFormatter) getLevelColor(level logrus.Level) func(format string, a ...interface{}) string {
	switch level {
	case logrus.DebugLevel:
		return color.New(color.FgHiBlack).SprintfFunc()
	case logrus.InfoLevel:
		return color.New(color.FgHiWhite).SprintfFunc()
	case logrus.WarnLevel:
		return color.New(color.FgYellow).SprintfFunc()
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		return color.New(color.FgRed).SprintfFunc()
	default:
		return color.New(color.FgWhite).SprintfFunc()
	}
}

// getServerColor returns the color function for a server name
func (f *ColoredFormatter) getServerColor(serverName string) func(format string, a ...interface{}) string {
	f.ColorMutex.RLock()
	colorFunc, ok := f.Colors[serverName]
	f.ColorMutex.RUnlock()

	if ok {
		return colorFunc
	}

	// Check if we've already assigned a color to this server
	f.ColorMutex.RLock()
	colorFunc, ok = f.ServerColors[serverName]
	f.ColorMutex.RUnlock()

	if ok {
		return colorFunc
	}

	// Assign a new color to this server
	f.ColorMutex.Lock()
	defer f.ColorMutex.Unlock()

	// Check again in case another goroutine assigned a color while we were waiting
	if colorFunc, ok := f.ServerColors[serverName]; ok {
		return colorFunc
	}

	// Assign a color based on the number of servers we've seen
	colorIndex := len(f.ServerColors) % len(availableColors)
	colorFunc = availableColors[colorIndex]
	f.ServerColors[serverName] = colorFunc

	return colorFunc
}

// ParseJSONLogEntry parses a JSON log entry into a logrus.Entry
func ParseJSONLogEntry(line string) (*logrus.Entry, error) {
	// Create a new entry
	entry := &logrus.Entry{
		Logger: logrus.New(),
		Data:   make(logrus.Fields),
	}

	// Parse the JSON
	if err := json.Unmarshal([]byte(line), &entry.Data); err != nil {
		return nil, err
	}

	// Extract timestamp
	if timestampStr, ok := entry.Data["timestamp"].(string); ok {
		timestamp, err := time.Parse(time.RFC3339Nano, timestampStr)
		if err != nil {
			// Try alternative formats
			timestamp, err = time.Parse("2006-01-02T15:04:05.000Z07:00", timestampStr)
			if err != nil {
				// Just use current time if we can't parse
				timestamp = time.Now()
			}
		}
		entry.Time = timestamp
		delete(entry.Data, "timestamp")
	} else {
		entry.Time = time.Now()
	}

	// Extract level
	if levelStr, ok := entry.Data["level"].(string); ok {
		level, err := logrus.ParseLevel(levelStr)
		if err != nil {
			level = logrus.InfoLevel
		}
		entry.Level = level
		delete(entry.Data, "level")
	} else {
		entry.Level = logrus.InfoLevel
	}

	// Extract message
	if msg, ok := entry.Data["message"].(string); ok {
		entry.Message = msg
		delete(entry.Data, "message")
	} else if msg, ok := entry.Data["msg"].(string); ok {
		entry.Message = msg
		delete(entry.Data, "msg")
	}

	return entry, nil
}
