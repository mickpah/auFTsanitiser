package main

import (
	"os"

	"auFTsanitiser/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args, os.Stdin, os.Stdout, os.Stderr))
}
