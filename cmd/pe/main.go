// Command pe is the CLI entry point. The router lives in internal/cli;
// this file just hands argv off and exits with the returned code.
package main

import (
	"os"

	"github.com/Shin-R2un/pe/internal/cli"
)

func main() {
	app := &cli.App{}
	os.Exit(app.Run(os.Args[1:]))
}
