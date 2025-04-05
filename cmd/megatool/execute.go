package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/megatool/internal/logging"
	"github.com/megatool/internal/utils"
)

// executeMcpServer executes an MCP server binary with logging
func executeMcpServer(serverName string, args []string, client string) error {
	// Parse the args to check for SSE mode flags
	var sseMode bool
	var port string = "8080"
	var baseURL string

	// Process args for SSE mode
	for i := 0; i < len(args); i++ {
		if args[i] == "--sse" {
			sseMode = true
		} else if args[i] == "--port" && i+1 < len(args) {
			port = args[i+1]
			i++ // Skip the next arg as we've consumed it
		} else if args[i] == "--base-url" && i+1 < len(args) {
			baseURL = args[i+1]
			i++ // Skip the next arg as we've consumed it
		}
	}

	// If base URL is not specified, construct it from the port
	if sseMode && baseURL == "" {
		baseURL = fmt.Sprintf("http://localhost:%s", port)
	}
	// Construct the binary name
	binaryName := "megatool-" + serverName

	// Find the binary
	binaryPath, err := utils.GetBinaryPath(binaryName)
	if err != nil {
		utils.PrintError("Server '%s' not found: %v", serverName, err)
		utils.PrintInfo("Run 'megatool run --help' to see available servers")
		return err
	}

	// Filter out SSE-related flags before passing to the server binary
	var filteredArgs []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--sse" {
			// Skip the --sse flag
			continue
		} else if args[i] == "--port" || args[i] == "--base-url" {
			// Skip the flag and its value
			i++
			continue
		} else {
			filteredArgs = append(filteredArgs, args[i])
		}
	}

	// Create a command to execute the server binary
	cmd := exec.Command(binaryPath, filteredArgs...)

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

	// Check if help flag is present
	helpMode := false
	for _, arg := range filteredArgs {
		if arg == "--help" || arg == "-h" {
			helpMode = true
			break
		}
	}

	// If SSE mode is enabled, we need to modify the command to run in SSE mode
	if sseMode {
		// Import the necessary packages
		utils.PrintInfo("Starting %s in SSE mode on %s", serverName, baseURL)

		// Add environment variables to tell the server to run in SSE mode
		cmd.Env = append(os.Environ(),
			"MCP_SERVER_MODE=sse",
			fmt.Sprintf("MCP_SERVER_PORT=%s", port),
			fmt.Sprintf("MCP_SERVER_BASE_URL=%s", baseURL))

		// If help mode is enabled, add an environment variable to disable logging
		if helpMode {
			cmd.Env = append(cmd.Env, "MCP_HELP_MODE=true")
		}
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		utils.PrintError("Failed to start %s: %v", binaryName, err)
		return err
	}

	// If help mode is enabled, don't set up logging
	if helpMode && sseMode {
		// Just use standard pipes for help mode
		go io.Copy(os.Stdout, stdoutPipe)
		go io.Copy(os.Stderr, stderrPipe)
	} else {
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
	}

	// Record the PID with client info if specified
	var opts utils.ServerRecordOptions
	if client != "" {
		opts.Client = client
	}

	if err := utils.AddServerRecord(serverName, cmd.Process.Pid, opts); err != nil {
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
