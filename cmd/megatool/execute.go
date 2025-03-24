package main

import (
	"io"
	"os"
	"os/exec"

	"github.com/megatool/internal/logging"
	"github.com/megatool/internal/utils"
)

// executeMcpServer executes an MCP server binary with logging
func executeMcpServer(serverName string, args []string) error {
	// Construct the binary name
	binaryName := "megatool-" + serverName

	// Find the binary
	binaryPath, err := utils.GetBinaryPath(binaryName)
	if err != nil {
		utils.PrintError("Server '%s' not found: %v", serverName, err)
		utils.PrintInfo("Run 'megatool run --help' to see available servers")
		return err
	}

	// Create a command to execute the server binary
	cmd := exec.Command(binaryPath, args...)

	// Set up stdout and stderr to capture output
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		utils.PrintError("Failed to create stdout pipe: %v", err)
		return err
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		utils.PrintError("Failed to create stderr pipe: %v", err)
		return err
	}

	// Connect stdin directly for MCP communication
	cmd.Stdin = os.Stdin

	// Start the command
	if err := cmd.Start(); err != nil {
		utils.PrintError("Failed to start %s: %v", binaryName, err)
		return err
	}

	// Now that we have the PID, set up logging
	logger, err := logging.NewLogger(serverName, cmd.Process.Pid)
	if err != nil {
		utils.PrintError("Failed to set up logging: %v", err)
		// Continue anyway, just use standard pipes
		go io.Copy(os.Stdout, stdoutPipe)
		go io.Copy(os.Stderr, stderrPipe)
	} else {
		// Log the server start
		logger.WithField("args", args).Info("Starting MCP server")

		// Get the log writer
		logWriter := logger.GetLogWriter()

		// Set up multi-writers for stdout and stderr
		go io.Copy(io.MultiWriter(os.Stdout, logWriter), stdoutPipe)
		go io.Copy(io.MultiWriter(os.Stderr, logWriter), stderrPipe)

		// Log the log file location
		utils.PrintInfo("Logs for %s (PID %d) will be written to: %s", 
			serverName, cmd.Process.Pid, logger.FilePath)
	}

	// Record the PID
	if err := utils.AddServerRecord(serverName, cmd.Process.Pid); err != nil {
		utils.PrintError("Failed to record server process: %v", err)
		// Continue anyway, this is not fatal
	}

	// Wait for the command to complete
	if err := cmd.Wait(); err != nil {
		// If the command returned an error, exit with the same code
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}

		// Otherwise, print the error
		utils.PrintError("Failed to execute %s: %v", binaryName, err)
		return err
	}

	return nil
}
