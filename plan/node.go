package plan

import (
	"strings"

	"github.com/permanu/docksmith/core"
	"github.com/permanu/docksmith/detect"
)

// staticNodeFrameworks produce static HTML output consumed by nginx, not a node server.
var staticNodeFrameworks = map[string]bool{
	"vite":             true,
	"create-react-app": true,
	"gatsby":           true,
	"angular":          true,
	"vue-cli":          true,
	"astro":            true,
}

func nodeCacheMount(pm string) string {
	switch pm {
	case "pnpm":
		return "/root/.local/share/pnpm/store"
	case "yarn":
		return "/usr/local/share/.cache/yarn"
	case "bun":
		return "/root/.bun/install/cache"
	default:
		return "/root/.npm"
	}
}

// pmLockfileArgs returns the file list for the deps COPY step.
func pmLockfileArgs(pm string) []string {
	switch pm {
	case "pnpm":
		return []string{"package.json", "pnpm-lock.yaml*", "./"}
	case "yarn":
		return []string{"package.json", "yarn.lock*", "./"}
	case "bun":
		return []string{"package.json", "bun.lockb*", "bun.lock*", "./"}
	default:
		return []string{"package.json", "package-lock.json*", "./"}
	}
}

// planNode builds a three-stage plan (deps -> build -> runtime).
// Static frameworks (vite, CRA, etc.) get nginx; server frameworks run the node process.
func planNode(fw *core.Framework) (*core.BuildPlan, error) {
	pm := fw.PackageManager
	if pm == "" {
		pm = "npm"
	}
	// If the user override (BuildCommand) explicitly uses a different package manager,
	// switch to that PM so the install step is consistent with the build step.
	if fw.BuildCommand != "" {
		switch {
		case strings.HasPrefix(fw.BuildCommand, "npm ") || strings.Contains(fw.BuildCommand, "npm install"):
			pm = "npm"
		case strings.HasPrefix(fw.BuildCommand, "yarn ") || strings.Contains(fw.BuildCommand, "yarn install"):
			pm = "yarn"
		case strings.HasPrefix(fw.BuildCommand, "pnpm ") || strings.Contains(fw.BuildCommand, "pnpm install"):
			pm = "pnpm"
		}
	}
	nodeImg := ResolveDockerTag("node", fw.NodeVersion)
	installCmd := detect.PMInstallCommand(pm)

	depsSteps := []core.Step{
		{Type: core.StepWorkdir, Args: []string{"/app"}},
	}
	if (pm == "pnpm" || pm == "yarn") && detect.NodeVersionAtLeast(fw.NodeVersion, 22) {
		// Node 22+ ships corepack but requires an explicit enable before pnpm/yarn work.
		depsSteps = append(depsSteps, core.Step{Type: core.StepRun, Args: []string{"corepack enable"}})
	}
	depsSteps = append(depsSteps,
		core.Step{Type: core.StepCopy, Args: pmLockfileArgs(pm)},
		core.Step{Type: core.StepRun, Args: []string{installCmd}, CacheMount: &core.CacheMount{Target: nodeCacheMount(pm)}},
	)
	depsStage := core.Stage{
		Name:  "deps",
		From:  nodeImg,
		Steps: depsSteps,
	}

	buildCmd := fw.BuildCommand
	if buildCmd == "" {
		buildCmd = detect.PMRunBuild(pm)
	}
	buildSteps := []core.Step{
		{Type: core.StepCopy, Args: []string{".", "."}},
		{Type: core.StepRun, Args: []string{buildCmd}},
	}
	buildStage := core.Stage{
		Name:  "build",
		From:  "deps",
		Steps: buildSteps,
	}

	var runtimeStage core.Stage
	if staticNodeFrameworks[fw.Name] {
		outputDir := fw.OutputDir
		if outputDir == "" {
			outputDir = "dist"
		}
		// Use absolute path relative to WORKDIR /app in the build stage.
		if !strings.HasPrefix(outputDir, "/") {
			outputDir = "/app/" + outputDir
		}
		runtimeStage = core.Stage{
			Name: "runtime",
			From: "nginx:alpine",
			Steps: []core.Step{
				{
					Type:     core.StepCopyFrom,
					CopyFrom: &core.CopyFrom{Stage: "build", Src: outputDir, Dst: "/usr/share/nginx/html"},
					Link:     true,
				},
			},
		}
		// nginx needs writable cache dirs before switching to non-root user.
		runtimeStage.Steps = append(runtimeStage.Steps, core.Step{
			Type: core.StepRun,
			Args: []string{
				"mkdir -p /var/cache/nginx/client_temp /var/cache/nginx/proxy_temp /var/cache/nginx/fastcgi_temp /var/cache/nginx/uwsgi_temp /var/cache/nginx/scgi_temp && " +
					"chown -R nginx:nginx /var/cache/nginx",
			},
		})
		addNonRootUser(&runtimeStage, "nginx")
		addHealthcheck(&runtimeStage, "static", 80)
	} else {
		startParts := strings.Fields(fw.StartCommand)
		if len(startParts) == 0 {
			startParts = strings.Fields(detect.PMRunStart(pm))
		}
		runtimeStage = core.Stage{
			Name: "runtime",
			From: nodeImg,
			Steps: []core.Step{
				{Type: core.StepWorkdir, Args: []string{"/app"}},
				{
					Type:     core.StepCopyFrom,
					CopyFrom: &core.CopyFrom{Stage: "build", Src: "/app", Dst: "/app"},
					Link:     true,
				},
				{Type: core.StepCmd, Args: startParts},
			},
		}
		addTini(&depsStage, &runtimeStage)
		addNonRootUser(&runtimeStage, "node")
		addHealthcheck(&runtimeStage, "node", fw.Port)
	}

	expose := fw.Port
	if staticNodeFrameworks[fw.Name] {
		expose = 80
	}

	return &core.BuildPlan{
		Framework:    fw.Name,
		Expose:       expose,
		Stages:       []core.Stage{depsStage, buildStage, runtimeStage},
		Dockerignore: []string{"node_modules", ".git"},
	}, nil
}
