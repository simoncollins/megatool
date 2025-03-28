package main

import (
	"fmt"
	"os"

	"github.com/megatool/internal/mcpserver"
)

func main() {
	// Create a new package version server
	packageVersionServer := NewPackageVersionServer()

	// Create and run the CLI app
	app := mcpserver.NewCliApp(packageVersionServer, nil, nil)
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
