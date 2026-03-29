// Command config-file shows how to load a docksmith.toml config file
// and use it to drive Dockerfile generation.
package main

import (
	"fmt"
	"os"

	"github.com/permanu/docksmith"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <project-dir>\n", os.Args[0])
		os.Exit(1)
	}
	dir := os.Args[1]

	// Load the config file (docksmith.toml, docksmith.yaml, or docksmith.json).
	// Returns nil if no config file exists.
	cfg, err := docksmith.LoadConfig(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}
	if cfg == nil {
		fmt.Fprintln(os.Stderr, "no docksmith config file found, using auto-detection")

		// Fall back to auto-detect without config overrides.
		dockerfile, fw, buildErr := docksmith.Build(dir)
		if buildErr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", buildErr)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "# Detected: %s\n", fw.Name)
		fmt.Print(dockerfile)
		return
	}

	fmt.Fprintf(os.Stderr, "# Config: runtime=%s\n", cfg.Runtime)

	// Convert config to a Framework for generation.
	fw := docksmith.ConfigToFramework(cfg)

	// Convert config fields to PlanOptions.
	opts, err := docksmith.ConfigToPlanOptions(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "options error: %v\n", err)
		os.Exit(1)
	}

	// Generate the Dockerfile.
	dockerfile, err := docksmith.GenerateDockerfile(fw, opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate error: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(dockerfile)
}
