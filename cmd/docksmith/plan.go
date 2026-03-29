package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/permanu/docksmith"
)

func runPlan(cfg config, args []string) {
	dir := targetDir(args)
	fw, err := docksmith.Detect(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if fw.Name == "dockerfile" {
		fmt.Fprintln(os.Stderr, "note: project has its own Dockerfile, no plan generated")
		os.Exit(0)
	}

	plan, err := docksmith.Plan(fw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	if cfg.isJSON() {
		data, err := json.MarshalIndent(plan, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(2)
		}
		fmt.Println(string(data))
		return
	}

	printPlan(plan)
}

func printPlan(plan *docksmith.BuildPlan) {
	fmt.Printf("framework: %s\n", plan.Framework)
	fmt.Printf("port:      %d\n", plan.Expose)
	fmt.Printf("stages:    %d\n", len(plan.Stages))
	for _, s := range plan.Stages {
		name := s.Name
		if name == "" {
			name = "(unnamed)"
		}
		fmt.Printf("  %-16s from=%-30s steps=%d\n", name, s.From, len(s.Steps))
	}
}
