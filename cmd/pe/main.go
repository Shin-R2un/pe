// Command pe is the CLI entry point. The router lives in internal/cli;
// this file just hands argv off and exits with the returned code.
package main

import (
	"os"

	"github.com/Shin-R2un/pe/internal/cli"
)

// version is the released build version. Overridden at link time by
// goreleaser via -ldflags "-X main.version=...". Unreleased builds keep
// the in-tree placeholder so `pe version` still prints something useful.
var version = "0.2.1-dev"

func main() {
	cli.Version = version
	os.Exit((&cli.App{}).Run(os.Args[1:]))
}
