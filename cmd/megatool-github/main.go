package main

import (
	"fmt"
	"os"

	"github.com/megatool/internal/mcpserver"
	"github.com/urfave/cli/v2"
)

func main() {
	// Create a new GitHub server
	githubServer := NewGitHubServer()

	// Define custom flags
	flags := []cli.Flag{
		&cli.BoolFlag{
			Name:    "configure",
			Aliases: []string{"c"},
			Usage:   "Configure GitHub MCP server",
		},
	}

	// Define custom action
	action := func(c *cli.Context) error {
		// Handle configuration mode
		if c.Bool("configure") {
			return githubServer.Configure()
		}

		// Load configuration
		if err := githubServer.LoadConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Run 'megatool github --configure' to configure the GitHub MCP server\n")
			return err
		}

		// Run the server
		return mcpserver.CreateAndRunServer(githubServer)
	}

	// Create and run the CLI app
	app := mcpserver.NewCliApp(githubServer, flags, action)
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
