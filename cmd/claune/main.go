package main

import (
	"fmt"
	"os"

	"github.com/everlier/claune/internal/cli"
)

var Version = "dev"

func main() {
	if err := cli.Run(os.Args[1:], Version); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

