package main

import (
	"fmt"
	"io"
	"os"

	"github.com/everlier/claune/internal/cli"
)

var Version = "dev"

var (
	runCLI func([]string, string) error = cli.Run
	stderr io.Writer                  = os.Stderr
	exit                              = os.Exit
)

func main() {
	if err := runCLI(os.Args[1:], Version); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		exit(1)
	}
}
