package main

import (
	"fmt"

	"github.com/megatool/internal/utils"
	"github.com/urfave/cli/v2"
)

// setupCommands returns the CLI commands for the application
func setupCommands() []*cli.Command {
	return []*cli.Command{
		logsCommand(),
		cleanupCommand(),
		{
			Name:  "install",
			Usage: "Install an MCP server into a client's configuration",
			Description: `Install an MCP server into a client's configuration.
Currently supports the following clients:
  - cline (Visual Studio Code Cline extension)`,
			ArgsUsage: "<server>",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "client",
					Aliases:  []string{"c"},
					Usage:    "Target MCP client (e.g., cline)",
					Required: true,
				},
			},
			Action: func(c *cli.Context) error {
				// Check if we have enough arguments
				if c.NArg() < 1 {
					// Show available servers
					fmt.Println("Available servers:")
					if err := listAvailableServers("  "); err != nil {
						fmt.Println("  No servers found")
					}
					return fmt.Errorf("no server specified")
				}

				// Get the client type
				clientStr := c.String("client")
				var clientType utils.ClientType
				switch clientStr {
				case "cline":
					clientType = utils.ClientCline
				default:
					utils.PrintError("Unsupported client type: %s", clientStr)
					utils.PrintInfo("Supported client types: cline")
					return fmt.Errorf("unsupported client type")
				}

				// Get the server name from the first argument
				serverName := c.Args().First()

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
					utils.PrintInfo("Run 'megatool ls' to see available servers")
					return fmt.Errorf("server not found")
				}

				// Get the config file path
				configPath, err := utils.GetClientConfigPath(clientType)
				if err != nil {
					utils.PrintError("Failed to get config path: %v", err)
					return err
				}

				// Install the server
				if err := utils.InstallServer(clientType, serverName); err != nil {
					utils.PrintError("Failed to install server: %v", err)
					return err
				}

				utils.PrintInfo("Server '%s' installed successfully into %s config at %s", serverName, clientStr, configPath)
				return nil
			},
			BashComplete: func(c *cli.Context) {
				// If we're completing the first argument, list available servers
				if c.NArg() == 0 {
					servers, err := utils.GetAvailableServers()
					if err != nil {
						return
					}
					for _, server := range servers {
						fmt.Println(server)
					}
				}
			},
		},
		{
			Name:  "run",
			Usage: "Run an MCP server",
			Description: `Run an MCP server with the specified name.
The server binary must be in the same directory as megatool or in the PATH.`,
			ArgsUsage: "<server>",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "configure",
					Usage: "Configure the server",
				},
				&cli.StringFlag{
					Name:  "client",
					Usage: "Target MCP client (e.g., cline)",
					Value: "",
				},
			},
			Action: func(c *cli.Context) error {
				// Check if we have enough arguments
				if c.NArg() < 1 {
					// Show available servers
					fmt.Println("Available servers:")
					if err := listAvailableServers("  "); err != nil {
						fmt.Println("  No servers found")
					}
					return fmt.Errorf("no server specified")
				}

				// Get the server name from the first argument
				serverName := c.Args().First()

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

				// Extract client if specified
				client := c.String("client")
				
				// Execute the specified MCP server
				// Pass all arguments after the server name
				return executeMcpServer(serverName, c.Args().Slice()[1:], client)
			},
			BashComplete: func(c *cli.Context) {
				// If we're completing the first argument, list available servers
				if c.NArg() == 0 {
					servers, err := utils.GetAvailableServers()
					if err != nil {
						return
					}
					for _, server := range servers {
						fmt.Println(server)
					}
				}
			},
		},
		{
			Name:  "ls",
			Usage: "List available MCP servers",
			Action: func(c *cli.Context) error {
				// Use the common function with no indentation
				err := listAvailableServers("")
				if err != nil {
					utils.PrintError("Failed to get available servers: %v", err)
					return err
				}
				return nil
			},
		},
		{
			Name:  "ps",
			Usage: "List running MCP servers",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "format",
					Aliases: []string{"f"},
					Value:   "table",
					Usage:   "Output format (table, json, csv)",
				},
				&cli.StringFlag{
					Name:    "fields",
					Value:   "name,pid,uptime,client",
					Usage:   "Comma-separated list of fields to display (name, pid, uptime, client)",
				},
				&cli.BoolFlag{
					Name:  "no-header",
					Usage: "Don't print header row",
				},
				&cli.StringFlag{
					Name:  "client",
					Usage: "Filter servers by client (e.g., cline)",
					Value: "",
				},
			},
			Action: func(c *cli.Context) error {
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
				
				// Filter by client if specified
				clientFilter := c.String("client")
				if clientFilter != "" {
					var filteredRecords []utils.ServerRecord
					for _, record := range records {
						if record.Client == clientFilter {
							filteredRecords = append(filteredRecords, record)
						}
					}
					records = filteredRecords
				}
				
				// Format and display records
				format := c.String("format")
				fields := c.String("fields")
				noHeader := c.Bool("no-header")
				
				return displayServerRecords(records, format, fields, !noHeader)
			},
		},
		{
			Name:      "stop",
			Usage:     "Stop a running MCP server",
			ArgsUsage: "<server>",
			Description: `Stop a running MCP server gracefully.
If multiple instances of the server are running, you must specify which one to stop
using the --pid flag, or use --all to stop all instances.`,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "all",
					Usage: "Stop all instances of the specified server",
				},
				&cli.IntFlag{
					Name:  "pid",
					Usage: "Stop a specific instance by PID",
					Value: 0,
				},
				&cli.StringFlag{
					Name:  "client",
					Usage: "Filter servers by client (e.g., cline)",
					Value: "",
				},
			},
			Action: func(c *cli.Context) error {
				// Get the server name from the first argument
				var serverName string
				if c.NArg() > 0 {
					serverName = c.Args().First()
				}
				
				// Get the PID flag
				pid := c.Int("pid")
				all := c.Bool("all")
				
				// Check if we have enough information
				if serverName == "" && pid == 0 {
					// Show running servers
					fmt.Println("Running servers:")
					
					// Get running servers
					records, err := utils.ReadServerRecords()
					if err != nil {
						fmt.Println("  Error reading server records")
						return fmt.Errorf("no server specified")
					}
					
					// Clean up stale records
					records = utils.CleanupStaleRecords(records)
					
					// Check if any servers are running
					if len(records) == 0 {
						fmt.Println("  No running MCP servers found")
						return fmt.Errorf("no server specified")
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
					
					return fmt.Errorf("no server specified")
				}
				
				// Read server records
				records, err := utils.ReadServerRecords()
				if err != nil {
					utils.PrintError("Failed to read server records: %v", err)
					return err
				}
				
				// Clean up stale records
				records = utils.CleanupStaleRecords(records)
				
				// Filter by client if specified
				clientFilter := c.String("client")
				if clientFilter != "" {
					var filteredRecords []utils.ServerRecord
					for _, record := range records {
						if record.Client == clientFilter {
							filteredRecords = append(filteredRecords, record)
						}
					}
					records = filteredRecords
				}
				
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
			},
			BashComplete: func(c *cli.Context) {
				// If we're completing the first argument, list running servers
				if c.NArg() == 0 {
					records, err := utils.ReadServerRecords()
					if err != nil {
						return
					}
					
					// Clean up stale records
					records = utils.CleanupStaleRecords(records)
					
					// Get unique server names
					serverNames := make(map[string]bool)
					for _, record := range records {
						serverNames[record.Name] = true
					}
					
					// Print each server name
					for name := range serverNames {
						fmt.Println(name)
					}
				}
			},
		},
	}
}

// setupHelpTemplates sets up custom help templates for the CLI
func setupHelpTemplates() {
	cli.AppHelpTemplate = `NAME:
   {{.Name}} - {{.Usage}}

USAGE:
   {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}
   {{if len .Authors}}
AUTHOR:
   {{range .Authors}}{{ . }}{{end}}
   {{end}}{{if .Commands}}
COMMANDS:
   logs        View MCP server logs
   cleanup     Clean up logs from MCP servers that are no longer running
   install     Install an MCP server into a client's configuration
   run         Run an MCP server
   ls          List available MCP servers
   ps          List running MCP servers
   stop        Stop a running MCP server
   help, h     Shows a list of commands or help for one command
{{end}}{{if .VisibleFlags}}
GLOBAL OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}
`

	cli.CommandHelpTemplate = `NAME:
   {{.HelpName}} - {{.Usage}}

USAGE:
   {{.HelpName}}{{if .VisibleFlags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}
   {{if .Category}}
CATEGORY:
   {{.Category}}
   {{end}}{{if .Description}}
DESCRIPTION:
   {{.Description}}
   {{end}}{{if .VisibleFlags}}
OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}
`
}
