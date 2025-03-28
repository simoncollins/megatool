package main

import (
	"fmt"
	"os"

	"github.com/megatool/internal/mcpserver"
)

func main() {
	// Create a new calculator server
	calculatorServer := NewCalculatorServer()

	// Create and run the CLI app
	app := mcpserver.NewCliApp(calculatorServer, nil, nil)
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
