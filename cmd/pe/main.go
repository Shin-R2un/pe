package main

import (
	"fmt"
	"os"
)

const version = "0.0.0"

func usage() {
	fmt.Fprintln(os.Stderr, `pe — paste-friendly snippet & phrase manager

Usage:
  pe add        register a snippet
  pe ls         list snippets
  pe find <q>   search snippets
  pe cp <id>    copy a snippet to the clipboard
  pe rm <id>    remove a snippet
  pe -v         print version`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "-v", "--version", "version":
		fmt.Println(version)
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "pe: subcommand %q not implemented yet\n", os.Args[1])
		os.Exit(2)
	}
}
