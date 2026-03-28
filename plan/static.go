package plan

import "github.com/permanu/docksmith/core"

// planStatic builds a single-stage BuildPlan for static sites served by nginx.
// The output directory (or the whole project when unspecified) is copied into
// nginx's default html root. No expose port is set because static sites don't
// bind a port themselves — nginx listens on 80 but the Validate() rules exempt
// the "static" framework from requiring Expose > 0.
func planStatic(fw *core.Framework) (*core.BuildPlan, error) {
	outputDir := fw.OutputDir
	if outputDir == "" {
		outputDir = "."
	}

	runtime := core.Stage{
		Name: "runtime",
		From: "nginx:alpine",
		Steps: []core.Step{
			{Type: core.StepCopy, Args: []string{outputDir, "/usr/share/nginx/html"}},
			{Type: core.StepExpose, Args: []string{"80"}},
			{Type: core.StepCmd, Args: []string{"nginx", "-g", "daemon off;"}},
		},
	}

	// nginx needs writable cache dirs before switching to non-root user.
	runtime.Steps = append(runtime.Steps, core.Step{
		Type: core.StepRun,
		Args: []string{
			"mkdir -p /var/cache/nginx/client_temp /var/cache/nginx/proxy_temp /var/cache/nginx/fastcgi_temp /var/cache/nginx/uwsgi_temp /var/cache/nginx/scgi_temp && " +
				"chown -R nginx:nginx /var/cache/nginx",
		},
	})
	addNonRootUser(&runtime, "nginx")
	addHealthcheck(&runtime, "static", 80)

	return &core.BuildPlan{
		Framework:    "static",
		Expose:       0,
		Stages:       []core.Stage{runtime},
		Dockerignore: []string{".git", "*.log", "node_modules"},
	}, nil
}
