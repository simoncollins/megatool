package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/megatool/internal/utils"
)

// Subcommand represents a megatool subcommand
type subcommand struct {
	name        string
	description string
	handler     func([]string) error
}

// Define available subcommands
var subcommands = []subcommand{
	{
		name:        "run",
		description: "Run an MCP server",
		handler:     runCommand,
	},
	{
		name:        "ls",
		description: "List available MCP servers",
		handler:     lsCommand,
	},
	// Future subcommands will be added here
	// {name: "ps", description: "List running MCP servers", handler: psCommand},
	// {name: "logs", description: "Show logs from MCP servers", handler: logsCommand},
}

// printUsage prints the main usage information
func printUsage() {
	fmt.Println("Usage: megatool <command> [args]")
	fmt.Println()
	fmt.Println("Available commands:")
	for _, cmd := range subcommands {
		fmt.Printf("  %-10s %s\n", cmd.name, cmd.description)
	}
	fmt.Println()
	fmt.Println("Run 'megatool <command> --help' for more information on a command.")
}

// listAvailableServers prints a list of available MCP servers with optional indentation
func listAvailableServers(indent string) error {
	// Get available servers
	servers, err := utils.GetAvailableServers()
	if err != nil {
		return err
	}

	// Check if any servers were found
	if len(servers) == 0 {
		fmt.Println(indent + "No MCP servers available")
		return nil
	}

	// Print each server on a separate line
	for _, server := range servers {
		fmt.Println(indent + server)
	}

	return nil
}

// printRunUsage prints usage information for the run command
func printRunUsage() {
	fmt.Println("Usage: megatool run <server> [flags]")
	fmt.Println()
	fmt.Println("Available servers:")

	// Use the common function with indentation
	err := listAvailableServers("  ")
	if err != nil {
		fmt.Println("  No servers found")
	}

	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --configure    - Configure the server")
	fmt.Println("  --help         - Show help for a server")
}

// lsCommand handles the 'ls' subcommand to list available MCP servers
func lsCommand(args []string) error {
	// Use the common function with no indentation
	err := listAvailableServers("")
	if err != nil {
		utils.PrintError("Failed to get available servers: %v", err)
		return err
	}
	
	return nil
}

// runCommand handles the 'run' subcommand
func runCommand(args []string) error {
	// Check if we have enough arguments
	if len(args) < 1 {
		printRunUsage()
		return fmt.Errorf("no server specified")
	}

	// Get the server name from the first argument
	serverName := args[0]

	// Handle help flag for run command
	if serverName == "--help" || serverName == "-h" {
		printRunUsage()
		return nil
	}

	// Check if the server exists in our list of available servers
	availableServers, err := utils.GetAvailableServers()
	if err != nil {
		utils.PrintError("Failed to get available servers: %v", err)
		return err
	}

	serverExists := false
	for _, server := range availableServers {
		if server == serverName {
			serverExists = true
			break
		}
	}

	if !serverExists {
		utils.PrintError("Server '%s' not found", serverName)
		utils.PrintInfo("Run 'megatool run --help' to see available servers")
		return fmt.Errorf("server not found")
	}

	// Execute the specified MCP server
	return executeMcpServer(serverName, args[1:])
}

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

	// Run the command
	err = cmd.Run()
	if err != nil {
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

func main() {
	// Check if we have enough arguments for a subcommand
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Get the subcommand name from the first argument
	cmdName := os.Args[1]

	// Handle help flag at the top level
	if cmdName == "--help" || cmdName == "-h" {
		printUsage()
		os.Exit(0)
	}

	// Find and execute the appropriate subcommand
	for _, cmd := range subcommands {
		if cmd.name == cmdName {
			err := cmd.handler(os.Args[2:])
			if err != nil {
				os.Exit(1)
			}
			return
		}
	}

	// If we get here, the subcommand was not recognized
	utils.PrintError("Unknown subcommand: %s", cmdName)
	utils.PrintInfo("Run 'megatool --help' to see available commands")
	os.Exit(1)
}
