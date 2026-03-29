// Command generate detects a project's framework and prints a production Dockerfile.
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

	dockerfile, fw, err := docksmith.Build(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if fw.Name == "dockerfile" {
		fmt.Fprintln(os.Stderr, "project already has a Dockerfile, skipping generation")
		os.Exit(0)
	}

	fmt.Fprintf(os.Stderr, "# Generated for %s (port %d)\n", fw.Name, fw.Port)
	fmt.Print(dockerfile)
}
