package main

import (
	"fmt"
	"os"

	"github.com/permanu/docksmith"
)

func runDockerfile(cfg config, args []string) {
	dir := targetDir(args)
	fw, err := docksmith.Detect(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if fw.Name == "dockerfile" {
		fmt.Fprintln(os.Stderr, "project already has a Dockerfile")
		os.Exit(0)
	}

	plan, err := docksmith.Plan(fw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	out := docksmith.EmitDockerfile(plan)
	if out == "" {
		fmt.Fprintln(os.Stderr, "error: emitter produced empty output")
		os.Exit(2)
	}

	fmt.Print(out)
}
