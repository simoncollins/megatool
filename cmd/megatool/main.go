package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"text/tabwriter"

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
	{
		name:        "ps",
		description: "List running MCP servers",
		handler:     psCommand,
	},
	{
		name:        "stop",
		description: "Stop a running MCP server",
		handler:     stopCommand,
	},
	// Future subcommands will be added here
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

// printPsUsage prints usage information for the ps command
func printPsUsage() {
	fmt.Println("Usage: megatool ps [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --format       - Output format (table, json, csv) [default: table]")
	fmt.Println("  --fields       - Comma-separated list of fields to display [default: name,pid,uptime]")
	fmt.Println("  --no-header    - Don't print header row")
	fmt.Println("  --help         - Show this help")
}

// psCommand handles the 'ps' subcommand to list running MCP servers
func psCommand(args []string) error {
	// Parse flags
	var format string
	var fields string
	var noHeader bool
	var help bool
	
	// Define flags
	psFlags := flag.NewFlagSet("ps", flag.ExitOnError)
	psFlags.StringVar(&format, "format", "table", "Output format (table, json, csv)")
	psFlags.StringVar(&fields, "fields", "name,pid,uptime", "Comma-separated list of fields to display")
	psFlags.BoolVar(&noHeader, "no-header", false, "Don't print header row")
	psFlags.BoolVar(&help, "help", false, "Show help")
	
	if err := psFlags.Parse(args); err != nil {
		return err
	}
	
	// Handle help flag
	if help || (psFlags.NArg() > 0 && psFlags.Arg(0) == "--help") {
		printPsUsage()
		return nil
	}
	
	// Read server records
	records, err := utils.ReadServerRecords()
	if err != nil {
		utils.PrintError("Failed to read server records: %v", err)
		return err
	}
	
	// Clean up stale records and save back
	activeRecords := utils.CleanupStaleRecords(records)
	if len(activeRecords) != len(records) {
		err = utils.WriteServerRecords(activeRecords)
		if err != nil {
			utils.PrintError("Failed to update server records: %v", err)
		}
		records = activeRecords
	}
	
	// Format and display records
	return displayServerRecords(records, format, fields, !noHeader)
}

// displayServerRecords formats and displays server records
func displayServerRecords(records []utils.ServerRecord, format, fields string, showHeader bool) error {
	// Split requested fields
	fieldList := strings.Split(fields, ",")
	
	// Check if we have any records
	if len(records) == 0 {
		fmt.Println("No running MCP servers found")
		return nil
	}
	
	// Generate output based on format
	switch format {
	case "table":
		return displayServerTable(records, fieldList, showHeader)
	case "json":
		return displayServerJSON(records, fieldList)
	case "csv":
		return displayServerCSV(records, fieldList, showHeader)
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
}

// displayServerTable displays server records in a table format
func displayServerTable(records []utils.ServerRecord, fields []string, showHeader bool) error {
	// Create a new tabwriter
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	
	// Print header if requested
	if showHeader {
		var headers []string
		for _, field := range fields {
			switch field {
			case "name":
				headers = append(headers, "NAME")
			case "pid":
				headers = append(headers, "PID")
			case "uptime":
				headers = append(headers, "UPTIME")
			default:
				headers = append(headers, strings.ToUpper(field))
			}
		}
		fmt.Fprintln(w, strings.Join(headers, "\t"))
	}
	
	// Count instances of each server
	serverCounts := make(map[string]int)
	for _, record := range records {
		serverCounts[record.Name]++
	}
	
	// Print each record
	for _, record := range records {
		var values []string
		for _, field := range fields {
			switch field {
			case "name":
				// Add instance indicator if there are multiple instances
				if serverCounts[record.Name] > 1 {
					values = append(values, fmt.Sprintf("%s (instance %d of %d)", 
						record.Name, 
						getInstanceNumber(records, record),
						serverCounts[record.Name]))
				} else {
					values = append(values, record.Name)
				}
			case "pid":
				values = append(values, fmt.Sprintf("%d", record.PID))
			case "uptime":
				values = append(values, utils.FormatUptime(record.StartTime))
			default:
				values = append(values, "N/A")
			}
		}
		fmt.Fprintln(w, strings.Join(values, "\t"))
	}
	
	return w.Flush()
}

// getInstanceNumber returns the instance number of a record among records with the same name
// sorted by start time (oldest first)
func getInstanceNumber(records []utils.ServerRecord, target utils.ServerRecord) int {
	// Filter records with the same name
	var sameNameRecords []utils.ServerRecord
	for _, record := range records {
		if record.Name == target.Name {
			sameNameRecords = append(sameNameRecords, record)
		}
	}
	
	// Sort by start time (oldest first)
	sort.Slice(sameNameRecords, func(i, j int) bool {
		return sameNameRecords[i].StartTime.Before(sameNameRecords[j].StartTime)
	})
	
	// Find the index of the target record
	for i, record := range sameNameRecords {
		if record.PID == target.PID {
			return i + 1
		}
	}
	
	return 0 // Should not happen
}

// displayServerJSON displays server records in JSON format
func displayServerJSON(records []utils.ServerRecord, fields []string) error {
	// Create a slice of maps for JSON output
	var result []map[string]interface{}
	
	// Count instances of each server
	serverCounts := make(map[string]int)
	for _, record := range records {
		serverCounts[record.Name]++
	}
	
	for _, record := range records {
		item := make(map[string]interface{})
		
		for _, field := range fields {
			switch field {
			case "name":
				// Add instance information if there are multiple instances
				if serverCounts[record.Name] > 1 {
					item["name"] = record.Name
					item["instance_number"] = getInstanceNumber(records, record)
					item["total_instances"] = serverCounts[record.Name]
				} else {
					item["name"] = record.Name
				}
			case "pid":
				item["pid"] = record.PID
			case "uptime":
				item["uptime"] = utils.FormatUptime(record.StartTime)
			}
		}
		
		result = append(result, item)
	}
	
	// Marshal to JSON
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	
	fmt.Println(string(jsonData))
	return nil
}

// printStopUsage prints usage information for the stop command
func printStopUsage() {
	fmt.Println("Usage: megatool stop <server> [flags]")
	fmt.Println()
	fmt.Println("Stop a running MCP server gracefully.")
	fmt.Println()
	fmt.Println("Arguments:")
	fmt.Println("  <server>      - Name of the MCP server to stop")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --all          - Stop all instances of the specified server")
	fmt.Println("  --pid <pid>    - Stop a specific instance by PID")
	fmt.Println("  --help         - Show this help")
	fmt.Println()
	fmt.Println("Running servers:")
	
	// Get running servers
	records, err := utils.ReadServerRecords()
	if err != nil {
		fmt.Println("  Error reading server records")
		return
	}
	
	// Clean up stale records
	records = utils.CleanupStaleRecords(records)
	
	// Check if any servers are running
	if len(records) == 0 {
		fmt.Println("  No running MCP servers found")
		return
	}
	
	// Count instances of each server
	serverCounts := make(map[string]int)
	for _, record := range records {
		serverCounts[record.Name]++
	}
	
	// Print each running server
	for _, record := range records {
		if serverCounts[record.Name] > 1 {
			fmt.Printf("  %s (instance %d of %d, PID: %d, Uptime: %s)\n", 
				record.Name,
				getInstanceNumber(records, record),
				serverCounts[record.Name],
				record.PID, 
				utils.FormatUptime(record.StartTime))
		} else {
			fmt.Printf("  %s (PID: %d, Uptime: %s)\n", 
				record.Name, 
				record.PID, 
				utils.FormatUptime(record.StartTime))
		}
	}
}

// stopCommand handles the 'stop' subcommand to terminate a running MCP server
func stopCommand(args []string) error {
	// Parse flags
	var all bool
	var pid int
	var help bool
	
	// Define flags
	stopFlags := flag.NewFlagSet("stop", flag.ExitOnError)
	stopFlags.BoolVar(&all, "all", false, "Stop all instances of the specified server")
	stopFlags.IntVar(&pid, "pid", 0, "Stop a specific instance by PID")
	stopFlags.BoolVar(&help, "help", false, "Show help")
	
	if err := stopFlags.Parse(args); err != nil {
		return err
	}
	
	// Handle help flag
	if help || len(args) == 0 || (stopFlags.NArg() == 0 && pid == 0) {
		printStopUsage()
		return nil
	}
	
	// Get the server name from the first non-flag argument
	var serverName string
	if stopFlags.NArg() > 0 {
		serverName = stopFlags.Arg(0)
	}
	
	// Read server records
	records, err := utils.ReadServerRecords()
	if err != nil {
		utils.PrintError("Failed to read server records: %v", err)
		return err
	}
	
	// Clean up stale records
	records = utils.CleanupStaleRecords(records)
	
	// Find matching server records
	var matchingRecords []utils.ServerRecord
	var remainingRecords []utils.ServerRecord
	
	for _, record := range records {
		if (serverName != "" && record.Name == serverName) || (pid > 0 && record.PID == pid) {
			matchingRecords = append(matchingRecords, record)
		} else {
			remainingRecords = append(remainingRecords, record)
		}
	}
	
	// Check if any matching servers were found
	if len(matchingRecords) == 0 {
		if serverName != "" {
			utils.PrintError("Server '%s' not found or not running", serverName)
		} else if pid > 0 {
			utils.PrintError("Process with PID %d not found or not an MCP server", pid)
		} else {
			utils.PrintError("No server specified")
		}
		
		if len(records) > 0 {
			utils.PrintInfo("Run 'megatool ps' to see running servers")
		}
		return fmt.Errorf("server not found")
	}
	
	// Handle multiple instances
	if len(matchingRecords) > 1 && !all && pid == 0 {
		utils.PrintInfo("Multiple instances of server '%s' are running:", serverName)
		for i, record := range matchingRecords {
			utils.PrintInfo("  %d. PID: %d, Uptime: %s", i+1, record.PID, utils.FormatUptime(record.StartTime))
		}
		utils.PrintInfo("Use --pid to specify which instance to stop, or --all to stop all instances")
		return fmt.Errorf("multiple instances found")
	}
	
	// Stop the matching servers
	stoppedCount := 0
	for _, record := range matchingRecords {
		if err := utils.TerminateProcess(record.PID); err != nil {
			utils.PrintError("Failed to stop server '%s' (PID: %d): %v", record.Name, record.PID, err)
			continue
		}
		stoppedCount++
		utils.PrintInfo("Server '%s' (PID: %d) stopped successfully", record.Name, record.PID)
	}
	
	// Update the server records
	if err := utils.WriteServerRecords(remainingRecords); err != nil {
		utils.PrintError("Failed to update server records: %v", err)
		// Continue anyway, this is not fatal
	}
	
	if stoppedCount > 0 && stoppedCount < len(matchingRecords) {
		utils.PrintInfo("Stopped %d of %d instances", stoppedCount, len(matchingRecords))
	} else if stoppedCount > 1 {
		utils.PrintInfo("All %d instances stopped successfully", stoppedCount)
	}
	
	return nil
}

// displayServerCSV displays server records in CSV format
func displayServerCSV(records []utils.ServerRecord, fields []string, showHeader bool) error {
	// Create a new CSV writer
	w := csv.NewWriter(os.Stdout)
	
	// Print header if requested
	if showHeader {
		var headers []string
		for _, field := range fields {
			switch field {
			case "name":
				headers = append(headers, "NAME")
			case "pid":
				headers = append(headers, "PID")
			case "uptime":
				headers = append(headers, "UPTIME")
			default:
				headers = append(headers, strings.ToUpper(field))
			}
		}
		if err := w.Write(headers); err != nil {
			return err
		}
	}
	
	// Count instances of each server
	serverCounts := make(map[string]int)
	for _, record := range records {
		serverCounts[record.Name]++
	}
	
	// Print each record
	for _, record := range records {
		var values []string
		for _, field := range fields {
			switch field {
			case "name":
				// Add instance indicator if there are multiple instances
				if serverCounts[record.Name] > 1 {
					values = append(values, fmt.Sprintf("%s (instance %d of %d)", 
						record.Name, 
						getInstanceNumber(records, record),
						serverCounts[record.Name]))
				} else {
					values = append(values, record.Name)
				}
			case "pid":
				values = append(values, fmt.Sprintf("%d", record.PID))
			case "uptime":
				values = append(values, utils.FormatUptime(record.StartTime))
			default:
				values = append(values, "N/A")
			}
		}
		if err := w.Write(values); err != nil {
			return err
		}
	}
	
	w.Flush()
	return w.Error()
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
