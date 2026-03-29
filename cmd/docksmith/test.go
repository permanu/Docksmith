package main

import (
	"fmt"
	"os"

	"github.com/permanu/docksmith"
)

func runTest(_ config, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: docksmith test <framework.yaml>")
		os.Exit(1)
	}

	yamlPath := args[0]
	results, err := docksmith.RunFrameworkTests(yamlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	passed := 0
	failed := 0
	for _, r := range results {
		if r.Passed {
			fmt.Printf("PASS  %s\n", r.Name)
			passed++
		} else {
			fmt.Printf("FAIL  %s: %s\n", r.Name, r.Reason)
			failed++
		}
	}

	fmt.Printf("\n%d passed, %d failed\n", passed, failed)
	if failed > 0 {
		os.Exit(1)
	}
}
