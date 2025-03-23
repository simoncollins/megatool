package main

import (
	"os"
	"os/exec"

	"github.com/megatool/internal/utils"
)

// executeMcpServer executes an MCP server binary
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

	// Connect stdin, stdout, and stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the command (instead of Run)
	if err := cmd.Start(); err != nil {
		utils.PrintError("Failed to start %s: %v", binaryName, err)
		return err
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
