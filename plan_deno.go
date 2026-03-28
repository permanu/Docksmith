package docksmith

import "strings"

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

func planDeno(fw *Framework) (*BuildPlan, error) {
	denoImg := ResolveDockerTag("deno", fw.DenoVersion)
	cacheTarget := denoCacheTarget(fw.StartCommand)

	buildSteps := []Step{
		{Type: StepWorkdir, Args: []string{"/app"}},
		{Type: StepEnv, Args: []string{"DENO_DIR", "/deno-dir"}},
		{Type: StepCopy, Args: []string{".", "."}},
		{Type: StepRun, Args: []string{"deno cache " + cacheTarget + " || true"}},
	}

	startParts := strings.Fields(fw.StartCommand)
	if len(startParts) == 0 {
		startParts = []string{"deno", "run", "-A", "main.ts"}
	}

	runtimeStage := Stage{
		Name: "runtime",
		From: denoImg,
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

	return &BuildPlan{
		Framework: fw.Name,
		Expose:    fw.Port,
		Stages: []Stage{
			{Name: "build", From: denoImg, Steps: buildSteps},
			runtimeStage,
		},
	}, nil
}
