package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

var version = "dev"

func main() {
	flag.Usage = usage
	formatFlag := flag.String("format", "text", "output format: text or json")
	quietFlag := flag.Bool("quiet", false, "suppress non-essential output")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	cfg := config{format: *formatFlag, quiet: *quietFlag}

	switch args[0] {
	case "detect":
		runDetect(cfg, args[1:])
	case "plan":
		runPlan(cfg, args[1:])
	case "dockerfile":
		runDockerfile(cfg, args[1:])
	case "build":
		runBuild(cfg, args[1:])
	case "eject":
		runEject(cfg, args[1:])
	case "init":
		runInit(cfg, args[1:])
	case "test":
		runTest(cfg, args[1:])
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
  docksmith [flags] <command> [args]

Commands:
  detect      detect framework in a directory
  plan        show build plan as human-readable or JSON
  dockerfile  emit Dockerfile to stdout
  build       detect, generate, and run docker build
  eject       write Dockerfile and .dockerignore to disk
  init        generate docksmith.toml template
  test        run inline YAML framework tests
  registry    search and install community framework definitions
  version     show version

Flags:`)
	flag.PrintDefaults()
}

type config struct {
	format string
	quiet  bool
}

func (c config) isJSON() bool { return c.format == "json" }

func targetDir(args []string) string {
	if len(args) > 0 && args[0] != "" {
		abs, err := filepath.Abs(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: resolve path: %v\n", err)
			os.Exit(1)
		}
		return abs
	}
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: get working directory: %v\n", err)
		os.Exit(1)
	}
	return dir
}
