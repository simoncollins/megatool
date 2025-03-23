package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:     "megatool",
		Usage:    "A tool for managing MCP servers",
		Commands: setupCommands(),
	}

	// Set up custom help templates
	setupHelpTemplates()

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
