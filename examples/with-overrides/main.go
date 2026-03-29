// Command with-overrides shows how to customize Dockerfile generation
// using PlanOption overrides.
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

	fw, err := docksmith.Detect(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "detect error: %v\n", err)
		os.Exit(1)
	}

	if fw.Name == "dockerfile" {
		fmt.Fprintln(os.Stderr, "project already has a Dockerfile")
		os.Exit(0)
	}

	// Override the generated Dockerfile with custom settings.
	// These take precedence over auto-detected values.
	dockerfile, err := docksmith.GenerateDockerfile(fw,
		docksmith.WithExpose(9090),                        // custom port
		docksmith.WithBuildCommand("npm run build:prod"),  // custom build step
		docksmith.WithStartCommand("node dist/server.js"), // custom start command
		docksmith.WithHealthcheck("curl -f http://localhost:9090/healthz || exit 1"),
		docksmith.WithUser("appuser"), // run as non-root user
		docksmith.WithExtraEnv(map[string]string{
			"NODE_ENV":  "production",
			"LOG_LEVEL": "info",
		}),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate error: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(dockerfile)
}
