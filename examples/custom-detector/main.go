// Command custom-detector shows how to register a custom framework detector.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/permanu/docksmith"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <project-dir>\n", os.Args[0])
		os.Exit(1)
	}

	// Register a custom detector that fires when it finds a "hugo.toml" file.
	// Custom detectors are prepended and checked before built-in ones.
	docksmith.RegisterDetector("hugo", func(dir string) *docksmith.Framework {
		if _, err := os.Stat(filepath.Join(dir, "hugo.toml")); err != nil {
			return nil // not a Hugo project
		}
		return &docksmith.Framework{
			Name:         "hugo",
			BuildCommand: "hugo --minify",
			StartCommand: "",
			Port:         1313,
			OutputDir:    "public",
		}
	})

	fw, err := docksmith.Detect(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Framework: %s\n", fw.Name)
	fmt.Printf("Port:      %d\n", fw.Port)
	if fw.OutputDir != "" {
		fmt.Printf("Output:    %s\n", fw.OutputDir)
	}
}
