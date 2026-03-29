package main

import (
	"flag"
	"fmt"
	"os"
)

var version = "dev"

func main() {
	flag.Usage = usage
	formatFlag := flag.String("format", "text", "output format: text or json")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	cfg := config{format: *formatFlag}

	switch args[0] {
	case "registry":
		runRegistry(cfg, args[1:])
	case "version":
		fmt.Printf("docksmith %s\n", version)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q\n\n", args[0])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `docksmith -- detect frameworks and generate production Dockerfiles

Usage:
  docksmith [--format json|text] <command> [args]

Commands:
  registry    search and install community framework definitions
  version     show version

Flags:`)
	flag.PrintDefaults()
}

type config struct {
	format string
}

func (c config) isJSON() bool { return c.format == "json" }
