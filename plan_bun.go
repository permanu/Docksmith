package docksmith

import "strings"

func planBun(fw *Framework) (*BuildPlan, error) {
	bunImg := ResolveDockerTag("bun", fw.BunVersion)

	buildSteps := []Step{
		{Type: StepWorkdir, Args: []string{"/app"}},
		{Type: StepCopy, Args: []string{"package.json", "bun.lockb*", "bun.lock*", "./"}},
		{Type: StepRun, Args: []string{"bun install --frozen-lockfile || bun install"}},
		{Type: StepCopy, Args: []string{".", "."}},
	}
	if fw.BuildCommand != "" {
		buildSteps = append(buildSteps, Step{
			Type: StepRun,
			Args: []string{fw.BuildCommand},
		})
	}

	startParts := strings.Fields(fw.StartCommand)
	if len(startParts) == 0 {
		startParts = []string{"bun", "run", "index.ts"}
	}

	runtimeStage := Stage{
		Name: "runtime",
		From: bunImg,
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
			{Name: "build", From: bunImg, Steps: buildSteps},
			runtimeStage,
		},
	}, nil
}
