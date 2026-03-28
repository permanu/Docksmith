package plan

import (
	"github.com/permanu/docksmith/core"
	"strings"
)

// denoCacheTarget picks the file to pass to `deno cache`.
// Falls back to main.ts when the last token of the start command is a task name,
// not a file (e.g. "deno task start").
func denoCacheTarget(startCommand string) string {
	parts := strings.Fields(startCommand)
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		if strings.HasSuffix(last, ".ts") || strings.HasSuffix(last, ".tsx") ||
			strings.HasSuffix(last, ".js") || strings.HasSuffix(last, ".jsx") {
			return last
		}
	}
	return "main.ts"
}

func planDeno(fw *core.Framework) (*core.BuildPlan, error) {
	denoImg := ResolveDockerTag("deno", fw.DenoVersion)
	cacheTarget := denoCacheTarget(fw.StartCommand)

	buildSteps := []core.Step{
		{Type: core.StepWorkdir, Args: []string{"/app"}},
		{Type: core.StepEnv, Args: []string{"DENO_DIR", "/deno-dir"}},
		{Type: core.StepCopy, Args: []string{".", "."}},
		{Type: core.StepRun, Args: []string{"deno cache " + cacheTarget + " || true"}},
	}

	startParts := strings.Fields(fw.StartCommand)
	if len(startParts) == 0 {
		startParts = []string{"deno", "run", "-A", "main.ts"}
	}

	runtimeStage := core.Stage{
		Name: "runtime",
		From: denoImg,
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

	addNonRootUser(&runtimeStage, "deno")
	addHealthcheck(&runtimeStage, "deno", fw.Port)

	return &core.BuildPlan{
		Framework: fw.Name,
		Expose:    fw.Port,
		Stages: []core.Stage{
			{Name: "build", From: denoImg, Steps: buildSteps},
			runtimeStage,
		},
	}, nil
}
