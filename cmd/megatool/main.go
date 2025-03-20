package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/megatool/internal/utils"
)

func printUsage() {
	fmt.Println("Usage: megatool <server> [flags]")
	fmt.Println()
	fmt.Println("Available servers:")

	// Try to get available servers
	servers, err := utils.GetAvailableServers()
	if err != nil || len(servers) == 0 {
		fmt.Println("  No servers found")
	} else {
		for _, server := range servers {
			fmt.Printf("  %s\n", server)
		}
	}

	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --configure    - Configure the server")
	fmt.Println("  --help         - Show help for a server")
}

func main() {
	// Check if we have enough arguments
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Get the server name from the first argument
	serverName := os.Args[1]

	// Handle help flag
	if serverName == "--help" || serverName == "-h" {
		printUsage()
		os.Exit(0)
	}

	// Construct the binary name
	binaryName := "megatool-" + serverName

	// Find the binary
	binaryPath, err := utils.GetBinaryPath(binaryName)
	if err != nil {
		utils.PrintError("Server '%s' not found: %v", serverName, err)
		utils.PrintInfo("Run 'megatool --help' to see available servers")
		os.Exit(1)
	}

	// Create a command to execute the server binary
	cmd := exec.Command(binaryPath, os.Args[2:]...)

	// Connect stdin, stdout, and stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	err = cmd.Run()
	if err != nil {
		// If the command returned an error, exit with the same code
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}

		// Otherwise, print the error and exit with code 1
		utils.PrintError("Failed to execute %s: %v", binaryName, err)
		os.Exit(1)
	}
}
