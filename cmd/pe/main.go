// Command pe is the CLI entry point. The router lives in internal/cli;
// this file just hands argv off and exits with the returned code.
package main

import (
	"os"
	"runtime/debug"
	"strings"

	"github.com/Shin-R2un/pe/internal/cli"
)

// version is overridden at link time by goreleaser via
// -ldflags "-X main.version=...". When missing (e.g. plain `go install`
// or local `make build`), resolveVersion falls back to the Go module
// version embedded by the toolchain so users still see "0.2.1" instead
// of "0.2.1-dev" after `go install ...@v0.2.1`.
var version = "0.2.4-dev"

func main() {
	cli.Version = resolveVersion(version)
	os.Exit((&cli.App{}).Run(os.Args[1:]))
}

func resolveVersion(linkTime string) string {
	// goreleaser builds inject a clean semver (no -dev suffix) — trust it.
	if linkTime != "" && !strings.HasSuffix(linkTime, "-dev") {
		return linkTime
	}
	// `go install github.com/.../@vX.Y.Z` embeds the module version as
	// VCS info. ReadBuildInfo surfaces it without needing -ldflags.
	if info, ok := debug.ReadBuildInfo(); ok {
		v := info.Main.Version
		if v != "" && v != "(devel)" {
			return strings.TrimPrefix(v, "v")
		}
	}
	return linkTime
}
