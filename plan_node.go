package docksmith

import "strings"

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

func planNode(fw *Framework) (*BuildPlan, error) {
	pm := fw.PackageManager
	if pm == "" {
		pm = "npm"
	}
	nodeImg := ResolveDockerTag("node", fw.NodeVersion)
	installCmd := pmInstallCommand(pm)

	depsSteps := []Step{
		{Type: StepWorkdir, Args: []string{"/app"}},
	}
	if (pm == "pnpm" || pm == "yarn") && nodeVersionAtLeast(fw.NodeVersion, 22) {
		// Node 22+ ships corepack but requires an explicit enable before pnpm/yarn work.
		depsSteps = append(depsSteps, Step{Type: StepRun, Args: []string{"corepack enable"}})
	}
	depsSteps = append(depsSteps,
		Step{Type: StepCopy, Args: pmLockfileArgs(pm)},
		Step{Type: StepRun, Args: []string{installCmd}, CacheMount: &CacheMount{Target: nodeCacheMount(pm)}},
	)
	depsStage := Stage{
		Name:  "deps",
		From:  nodeImg,
		Steps: depsSteps,
	}

	buildCmd := fw.BuildCommand
	if buildCmd == "" {
		buildCmd = pmRunBuild(pm)
	}
	buildSteps := []Step{
		{Type: StepCopy, Args: []string{".", "."}},
		{Type: StepRun, Args: []string{buildCmd}},
	}
	buildStage := Stage{
		Name:  "build",
		From:  "deps",
		Steps: buildSteps,
	}

	var runtimeStage Stage
	if staticNodeFrameworks[fw.Name] {
		outputDir := fw.OutputDir
		if outputDir == "" {
			outputDir = "dist"
		}
		runtimeStage = Stage{
			Name: "runtime",
			From: "nginx:alpine",
			Steps: []Step{
				{
					Type:     StepCopyFrom,
					CopyFrom: &CopyFrom{Stage: "build", Src: outputDir, Dst: "/usr/share/nginx/html"},
					Link:     true,
				},
			},
		}
	} else {
		startParts := strings.Fields(fw.StartCommand)
		if len(startParts) == 0 {
			startParts = strings.Fields(pmRunStart(pm))
		}
		runtimeStage = Stage{
			Name: "runtime",
			From: nodeImg,
			Steps: []Step{
				{Type: StepWorkdir, Args: []string{"/app"}},
				{
					Type:     StepCopyFrom,
					CopyFrom: &CopyFrom{Stage: "build", Src: "/app", Dst: "/app"},
					Link:     true,
				},
				{Type: StepCmd, Args: startParts},
			},
		}
	}

	expose := fw.Port
	if staticNodeFrameworks[fw.Name] {
		expose = 80
	}

	return &BuildPlan{
		Framework:    fw.Name,
		Expose:       expose,
		Stages:       []Stage{depsStage, buildStage, runtimeStage},
		Dockerignore: []string{"node_modules", ".git"},
	}, nil
}
