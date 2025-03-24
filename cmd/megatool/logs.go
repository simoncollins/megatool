package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hpcloud/tail"
	"github.com/megatool/internal/logging"
	"github.com/megatool/internal/utils"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// LogEntry represents a parsed log entry with timestamp for sorting
type LogEntry struct {
	ServerName string
	PID        int
	Entry      *logrus.Entry
	Line       string
}

// logsCommand returns the logs command
func logsCommand() *cli.Command {
	return &cli.Command{
		Name:    "logs",
		Aliases: []string{"log"},
		Usage:   "View MCP server logs",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "follow",
				Aliases: []string{"f"},
				Usage:   "Follow log output",
			},
			&cli.IntFlag{
				Name:    "lines",
				Aliases: []string{"n"},
				Usage:   "Number of lines to show",
				Value:   20,
			},
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "Show logs for all servers, not just active ones",
			},
		},
		ArgsUsage: "[server]",
		Action:    logsAction,
	}
}

// logsAction handles the logs command
func logsAction(c *cli.Context) error {
	// Get server name from arguments
	var serverName string
	if c.NArg() > 0 {
		serverName = c.Args().First()
	}

	// Get options
	follow := c.Bool("follow")
	lines := c.Int("lines")
	showAll := c.Bool("all")

	// Get active servers
	activeServers := make(map[string]bool)
	if !showAll {
		records, err := utils.ReadServerRecords()
		if err != nil {
			utils.PrintError("Failed to read server records: %v", err)
			return err
		}

		// Clean up stale records
		records = utils.CleanupStaleRecords(records)

		// Add active servers to map
		for _, record := range records {
			activeServers[record.Name] = true
		}

		// If no active servers and a specific server was requested, show an error
		if len(activeServers) == 0 && serverName != "" {
			utils.PrintError("No active MCP servers found")
			return fmt.Errorf("no active MCP servers")
		}
	}

	// If following logs, use tail
	if follow {
		return followLogs(serverName, activeServers, showAll)
	}

	// Otherwise, show the last N lines
	return showLogs(serverName, lines, activeServers, showAll)
}

// showLogs shows the last N lines of logs
func showLogs(serverName string, lines int, activeServers map[string]bool, showAll bool) error {
	// Get log files
	logFiles, err := getLogFiles(serverName, activeServers, showAll)
	if err != nil {
		return err
	}

	if len(logFiles) == 0 {
		if serverName != "" {
			utils.PrintError("No logs found for server '%s'", serverName)
		} else {
			utils.PrintError("No logs found")
		}
		return nil
	}

	// Create a colored formatter for output
	formatter := logging.NewColoredFormatter()

	// Read the last N lines from each log file
	var entries []LogEntry
	for _, logFile := range logFiles {
		lastLines, err := readLastLines(logFile.Path, lines)
		if err != nil {
			utils.PrintError("Failed to read log file %s: %v", logFile.Path, err)
			continue
		}

		// Parse each line
		for _, line := range lastLines {
			entry, err := logging.ParseJSONLogEntry(line)
			if err != nil {
				// If we can't parse as JSON, just show the raw line
				fmt.Println(line)
				continue
			}

			// Add server name and PID if not present
			if _, ok := entry.Data["server"]; !ok {
				entry.Data["server"] = logFile.ServerName
			}
			if _, ok := entry.Data["pid"]; !ok {
				entry.Data["pid"] = logFile.PID
			}

			entries = append(entries, LogEntry{
				ServerName: logFile.ServerName,
				PID:        logFile.PID,
				Entry:      entry,
				Line:       line,
			})
		}
	}

	// Sort entries by timestamp
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Entry.Time.Before(entries[j].Entry.Time)
	})

	// Print entries
	for _, entry := range entries {
		formatted, err := formatter.Format(entry.Entry)
		if err != nil {
			fmt.Println(entry.Line)
			continue
		}
		fmt.Print(string(formatted))
	}

	return nil
}

// followLogs follows logs in real-time
func followLogs(serverName string, activeServers map[string]bool, showAll bool) error {
	// Get log files
	logFiles, err := getLogFiles(serverName, activeServers, showAll)
	if err != nil {
		return err
	}

	if len(logFiles) == 0 {
		if serverName != "" {
			utils.PrintError("No logs found for server '%s'", serverName)
		} else {
			utils.PrintError("No logs found")
		}
		return nil
	}

	// Create a colored formatter for output
	formatter := logging.NewColoredFormatter()

	// Create a channel for log entries
	entryChan := make(chan LogEntry)

	// Start tailing each log file
	for _, logFile := range logFiles {
		go func(lf LogFile) {
			t, err := tail.TailFile(lf.Path, tail.Config{
				Follow:    true,
				ReOpen:    true,
				MustExist: true,
			})
			if err != nil {
				utils.PrintError("Failed to tail log file %s: %v", lf.Path, err)
				return
			}

			for line := range t.Lines {
				entry, err := logging.ParseJSONLogEntry(line.Text)
				if err != nil {
					// If we can't parse as JSON, create a simple entry
					entry = &logrus.Entry{
						Logger:  logrus.New(),
						Data:    make(logrus.Fields),
						Time:    time.Now(),
						Level:   logrus.InfoLevel,
						Message: line.Text,
					}
				}

				// Add server name and PID if not present
				if _, ok := entry.Data["server"]; !ok {
					entry.Data["server"] = lf.ServerName
				}
				if _, ok := entry.Data["pid"]; !ok {
					entry.Data["pid"] = lf.PID
				}

				entryChan <- LogEntry{
					ServerName: lf.ServerName,
					PID:        lf.PID,
					Entry:      entry,
					Line:       line.Text,
				}
			}
		}(logFile)
	}

	// Print entries as they come in
	fmt.Println("Following logs. Press Ctrl+C to exit.")
	for entry := range entryChan {
		formatted, err := formatter.Format(entry.Entry)
		if err != nil {
			fmt.Println(entry.Line)
			continue
		}
		fmt.Print(string(formatted))
	}

	return nil
}

// LogFile represents a log file with server name and PID
type LogFile struct {
	Path       string
	ServerName string
	PID        int
}

// getLogFiles returns a list of log files
func getLogFiles(serverName string, activeServers map[string]bool, showAll bool) ([]LogFile, error) {
	// Get log directory
	logDir, err := logging.GetLogDirectory()
	if err != nil {
		return nil, err
	}

	var logFiles []LogFile

	// If server name is provided, only look in that server's directory
	if serverName != "" {
		// Check if server is active
		if !showAll && !activeServers[serverName] {
			utils.PrintError("Server '%s' is not active", serverName)
			return nil, fmt.Errorf("server not active")
		}

		serverDir := filepath.Join(logDir, serverName)
		if _, err := os.Stat(serverDir); os.IsNotExist(err) {
			return nil, fmt.Errorf("no logs found for server '%s'", serverName)
		}

		// Get all log files in the server directory
		files, err := os.ReadDir(serverDir)
		if err != nil {
			return nil, err
		}

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".log") {
				continue
			}

			// Extract PID from filename
			var pid int
			_, err := fmt.Sscanf(file.Name(), "server_%d.log", &pid)
			if err != nil {
				continue
			}

			// Check if server is active
			if !showAll && !utils.IsProcessRunning(pid) {
				continue
			}

			logFiles = append(logFiles, LogFile{
				Path:       filepath.Join(serverDir, file.Name()),
				ServerName: serverName,
				PID:        pid,
			})
		}
	} else {
		// Look in all server directories
		serverDirs, err := os.ReadDir(logDir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, err
		}

		for _, serverDir := range serverDirs {
			if !serverDir.IsDir() {
				continue
			}

			// Check if server is active
			if !showAll && !activeServers[serverDir.Name()] {
				continue
			}

			// Get all log files in the server directory
			files, err := os.ReadDir(filepath.Join(logDir, serverDir.Name()))
			if err != nil {
				continue
			}

			for _, file := range files {
				if file.IsDir() || !strings.HasSuffix(file.Name(), ".log") {
					continue
				}

				// Extract PID from filename
				var pid int
				_, err := fmt.Sscanf(file.Name(), "server_%d.log", &pid)
				if err != nil {
					continue
				}

				// Check if server is active
				if !showAll && !utils.IsProcessRunning(pid) {
					continue
				}

				logFiles = append(logFiles, LogFile{
					Path:       filepath.Join(logDir, serverDir.Name(), file.Name()),
					ServerName: serverDir.Name(),
					PID:        pid,
				})
			}
		}
	}

	return logFiles, nil
}

// readLastLines reads the last n lines from a file
func readLastLines(filePath string, n int) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Create a ring buffer to hold the last n lines
	lines := make([]string, n)
	lineCount := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines[lineCount%n] = scanner.Text()
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// If we read fewer than n lines, return just those lines
	if lineCount < n {
		return lines[:lineCount], nil
	}

	// Otherwise, rearrange the ring buffer to return the last n lines in order
	result := make([]string, n)
	for i := 0; i < n; i++ {
		result[i] = lines[(lineCount+i)%n]
	}

	return result, nil
}
