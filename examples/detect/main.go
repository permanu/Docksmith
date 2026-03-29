// Command detect shows how to detect the framework for a project directory.
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
	fw, err := docksmith.Detect(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Framework: %s\n", fw.Name)
	fmt.Printf("Port:      %d\n", fw.Port)
	if fw.BuildCommand != "" {
		fmt.Printf("Build:     %s\n", fw.BuildCommand)
	}
	if fw.StartCommand != "" {
		fmt.Printf("Start:     %s\n", fw.StartCommand)
	}
}
