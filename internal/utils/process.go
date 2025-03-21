package utils

import (
	"fmt"
	"os"
	"syscall"
)

// TerminateProcess sends a SIGTERM signal to gracefully terminate a process
func TerminateProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process not found: %w", err)
	}

	// Send SIGTERM for graceful termination
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to terminate process: %w", err)
	}

	return nil
}

// IsProcessRunning checks if a process with the given PID is still running
func IsProcessRunning(pid int) bool {
	// On Unix-like systems, we can send signal 0 to check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds, so we need to send a signal to check
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
