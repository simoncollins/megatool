package utils

import (
	"os"
	"syscall"
)

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
