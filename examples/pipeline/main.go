// Command pipeline demonstrates each stage of docksmith separately:
// detect -> plan -> emit.
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

	// Stage 1: Detect the framework.
	fw, err := docksmith.Detect(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "detect failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "[detect] framework=%s port=%d\n", fw.Name, fw.Port)

	if fw.Name == "dockerfile" {
		fmt.Fprintln(os.Stderr, "[detect] project has a Dockerfile, nothing to generate")
		os.Exit(0)
	}

	// Stage 2: Convert the Framework into an abstract BuildPlan.
	bp, err := docksmith.Plan(fw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "plan failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "[plan]   stages=%d expose=%d\n", len(bp.Stages), bp.Expose)
	for i, stage := range bp.Stages {
		fmt.Fprintf(os.Stderr, "         stage[%d] name=%q from=%q steps=%d\n",
			i, stage.Name, stage.From, len(stage.Steps))
	}

	// Stage 3: Emit the Dockerfile from the plan.
	dockerfile := docksmith.EmitDockerfile(bp)
	fmt.Fprintf(os.Stderr, "[emit]   %d bytes\n", len(dockerfile))

	// Also generate a .dockerignore.
	dockerignore := docksmith.GenerateDockerignore(fw)
	fmt.Fprintf(os.Stderr, "[emit]   dockerignore: %d bytes\n", len(dockerignore))

	fmt.Print(dockerfile)
}
