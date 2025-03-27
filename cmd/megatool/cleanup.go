package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/megatool/internal/logging"
	"github.com/megatool/internal/utils"
	"github.com/urfave/cli/v2"
)

// cleanupCommand returns the cleanup command
func cleanupCommand() *cli.Command {
	return &cli.Command{
		Name:  "cleanup",
		Usage: "Clean up logs from MCP servers that are no longer running",
		Description: `Clean up logs from MCP servers that are no longer running.
This command will:
1. Remove log files for processes that are no longer running
2. Remove entire server log directories if all logs are older than the specified threshold
3. Clean up stale server records from the running-servers.json file`,
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "days",
				Aliases: []string{"d"},
				Usage:   "Remove logs older than this many days",
				Value:   30,
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Show what would be deleted without actually deleting",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Skip confirmation prompts",
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Show detailed information about the cleanup process",
			},
		},
		Action: cleanupAction,
	}
}

// cleanupAction handles the cleanup command
func cleanupAction(c *cli.Context) error {
	// Get options
	days := c.Int("days")
	dryRun := c.Bool("dry-run")
	force := c.Bool("force")
	verbose := c.Bool("verbose")

	// Clean up server records first
	if err := cleanupServerRecords(verbose, dryRun); err != nil {
		return err
	}

	// Clean up log directories
	return cleanupLogDirectories(days, verbose, dryRun, force)
}

// cleanupServerRecords cleans up stale server records
func cleanupServerRecords(verbose, dryRun bool) error {
	// Read server records
	records, err := utils.ReadServerRecords()
	if err != nil {
		utils.PrintError("Failed to read server records: %v", err)
		return err
	}

	// Get the original count
	originalCount := len(records)

	// Clean up stale records
	activeRecords := utils.CleanupStaleRecords(records)
	removedCount := originalCount - len(activeRecords)

	// Print results
	if verbose || removedCount > 0 {
		if removedCount > 0 {
			utils.PrintInfo("Found %d stale server records", removedCount)
		} else {
			utils.PrintInfo("No stale server records found")
		}
	}

	// Write back the cleaned records if not in dry-run mode and if records were removed
	if !dryRun && removedCount > 0 {
		if err := utils.WriteServerRecords(activeRecords); err != nil {
			utils.PrintError("Failed to update server records: %v", err)
			return err
		}
		utils.PrintInfo("Removed %d stale server records", removedCount)
	} else if dryRun && removedCount > 0 {
		utils.PrintInfo("Would remove %d stale server records (dry run)", removedCount)
	}

	return nil
}

// cleanupLogDirectories cleans up log directories
func cleanupLogDirectories(days int, verbose, dryRun, force bool) error {
	// Get log directory
	logDir, err := logging.GetLogDirectory()
	if err != nil {
		utils.PrintError("Failed to get log directory: %v", err)
		return err
	}

	// Check if log directory exists
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		if verbose {
			utils.PrintInfo("Log directory does not exist: %s", logDir)
		}
		return nil
	}

	// Get all server directories
	serverDirs, err := os.ReadDir(logDir)
	if err != nil {
		utils.PrintError("Failed to read log directory: %v", err)
		return err
	}

	if len(serverDirs) == 0 {
		if verbose {
			utils.PrintInfo("No server log directories found")
		}
		return nil
	}

	// Compile regex for log files
	logFileRegex := regexp.MustCompile(`^server_(\d+)\.log$`)

	// Calculate threshold time
	threshold := time.Now().AddDate(0, 0, -days)

	// Track statistics
	var (
		totalDirsRemoved    int
		totalFilesRemoved   int
		totalBytesRemoved   int64
		dirsToRemove        []string
		filesToRemove       []string
		activeDirs          []string
		activeFiles         []string
		oldestActiveLogTime time.Time
	)

	// Process each server directory
	for _, serverDir := range serverDirs {
		if !serverDir.IsDir() {
			continue
		}

		serverName := serverDir.Name()
		serverPath := filepath.Join(logDir, serverName)

		if verbose {
			utils.PrintInfo("Checking server directory: %s", serverName)
		}

		// Get all log files in the server directory
		files, err := os.ReadDir(serverPath)
		if err != nil {
			utils.PrintError("Failed to read server directory %s: %v", serverName, err)
			continue
		}

		// Track active processes and log files for this server
		hasActiveProcess := false
		var serverLogFiles []string
		var serverActiveLogFiles []string
		var serverInactiveLogFiles []string
		var newestLogTime time.Time

		// Check each log file
		for _, file := range files {
			if file.IsDir() {
				continue
			}

			fileName := file.Name()
			filePath := filepath.Join(serverPath, fileName)

			// Check if it's a log file
			matches := logFileRegex.FindStringSubmatch(fileName)
			if len(matches) < 2 {
				// Handle compressed log files (e.g., server_1234.log.gz)
				if strings.HasPrefix(fileName, "server_") && strings.HasSuffix(fileName, ".log.gz") {
					// Extract PID from compressed log filename
					pidStr := strings.TrimPrefix(fileName, "server_")
					pidStr = strings.TrimSuffix(pidStr, ".log.gz")
					
					if pid, err := strconv.Atoi(pidStr); err == nil {
						// Check if process is running
						if utils.IsProcessRunning(pid) {
							hasActiveProcess = true
							serverActiveLogFiles = append(serverActiveLogFiles, filePath)
							if verbose {
								utils.PrintInfo("  Found active compressed log file: %s (PID: %d)", fileName, pid)
							}
						} else {
							serverInactiveLogFiles = append(serverInactiveLogFiles, filePath)
							if verbose {
								utils.PrintInfo("  Found inactive compressed log file: %s (PID: %d)", fileName, pid)
							}
						}
					}
				}
				continue
			}

			// Extract PID from filename
			pidStr := matches[1]
			pid, err := strconv.Atoi(pidStr)
			if err != nil {
				continue
			}

			// Get file info for timestamp
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				utils.PrintError("Failed to get file info for %s: %v", filePath, err)
				continue
			}

			// Update newest log time
			modTime := fileInfo.ModTime()
			if modTime.After(newestLogTime) {
				newestLogTime = modTime
			}

			// Add to server log files
			serverLogFiles = append(serverLogFiles, filePath)

			// Check if process is running
			if utils.IsProcessRunning(pid) {
				hasActiveProcess = true
				serverActiveLogFiles = append(serverActiveLogFiles, filePath)
				if verbose {
					utils.PrintInfo("  Found active log file: %s (PID: %d)", fileName, pid)
				}
			} else {
				serverInactiveLogFiles = append(serverInactiveLogFiles, filePath)
				if verbose {
					utils.PrintInfo("  Found inactive log file: %s (PID: %d)", fileName, pid)
				}
			}
		}

		// Determine what to do with this server directory
		if hasActiveProcess {
			// Server has active processes, keep directory but remove inactive log files
			activeDirs = append(activeDirs, serverPath)
			activeFiles = append(activeFiles, serverActiveLogFiles...)
			filesToRemove = append(filesToRemove, serverInactiveLogFiles...)
			
			// Update oldest active log time for reporting
			if oldestActiveLogTime.IsZero() || newestLogTime.Before(oldestActiveLogTime) {
				oldestActiveLogTime = newestLogTime
			}
			
			if verbose {
				utils.PrintInfo("  Server %s has active processes, keeping directory", serverName)
				if len(serverInactiveLogFiles) > 0 {
					utils.PrintInfo("  Will remove %d inactive log files", len(serverInactiveLogFiles))
				}
			}
		} else if len(serverLogFiles) == 0 {
			// Empty directory, remove it
			dirsToRemove = append(dirsToRemove, serverPath)
			if verbose {
				utils.PrintInfo("  Server %s has no log files, will remove directory", serverName)
			}
		} else if newestLogTime.Before(threshold) {
			// All logs are older than threshold, remove directory
			dirsToRemove = append(dirsToRemove, serverPath)
			if verbose {
				utils.PrintInfo("  Server %s has no active processes and all logs are older than %d days, will remove directory", serverName, days)
				utils.PrintInfo("  Newest log is from %s", newestLogTime.Format("2006-01-02 15:04:05"))
			}
		} else {
			// No active processes but logs are newer than threshold, keep directory but remove inactive log files
			activeDirs = append(activeDirs, serverPath)
			filesToRemove = append(filesToRemove, serverInactiveLogFiles...)
			if verbose {
				utils.PrintInfo("  Server %s has no active processes but logs are newer than %d days, keeping directory", serverName, days)
				utils.PrintInfo("  Newest log is from %s", newestLogTime.Format("2006-01-02 15:04:05"))
				if len(serverInactiveLogFiles) > 0 {
					utils.PrintInfo("  Will remove %d inactive log files", len(serverInactiveLogFiles))
				}
			}
		}
	}

	// Calculate total bytes to be removed
	for _, filePath := range filesToRemove {
		fileInfo, err := os.Stat(filePath)
		if err == nil {
			totalBytesRemoved += fileInfo.Size()
		}
	}

	for _, dirPath := range dirsToRemove {
		dirSize, err := getDirSize(dirPath)
		if err == nil {
			totalBytesRemoved += dirSize
		}
	}

	// Print summary
	totalDirsRemoved = len(dirsToRemove)
	totalFilesRemoved = len(filesToRemove)

	if totalDirsRemoved == 0 && totalFilesRemoved == 0 {
		utils.PrintInfo("No logs to clean up")
		return nil
	}

	utils.PrintInfo("Cleanup summary:")
	if totalDirsRemoved > 0 {
		utils.PrintInfo("  Directories to remove: %d", totalDirsRemoved)
	}
	if totalFilesRemoved > 0 {
		utils.PrintInfo("  Log files to remove: %d", totalFilesRemoved)
	}
	utils.PrintInfo("  Total space to be freed: %s", formatBytes(totalBytesRemoved))

	// Ask for confirmation if not in force mode and not in dry-run mode
	if !force && !dryRun {
		if !confirmAction("Do you want to proceed with cleanup?") {
			utils.PrintInfo("Cleanup cancelled")
			return nil
		}
	}

	// Perform cleanup if not in dry-run mode
	if !dryRun {
		// Remove files first
		for _, filePath := range filesToRemove {
			if err := os.Remove(filePath); err != nil {
				utils.PrintError("Failed to remove file %s: %v", filePath, err)
			} else if verbose {
				utils.PrintInfo("Removed file: %s", filePath)
			}
		}

		// Then remove directories
		for _, dirPath := range dirsToRemove {
			if err := os.RemoveAll(dirPath); err != nil {
				utils.PrintError("Failed to remove directory %s: %v", dirPath, err)
			} else if verbose {
				utils.PrintInfo("Removed directory: %s", dirPath)
			}
		}

		utils.PrintInfo("Cleanup completed successfully")
		utils.PrintInfo("Removed %d directories and %d log files", totalDirsRemoved, totalFilesRemoved)
		utils.PrintInfo("Freed up %s of disk space", formatBytes(totalBytesRemoved))
	} else {
		utils.PrintInfo("Dry run completed. No files were actually removed.")
	}

	return nil
}

// getDirSize returns the total size of all files in a directory
func getDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// formatBytes formats bytes into a human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// confirmAction asks the user for confirmation
func confirmAction(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}
