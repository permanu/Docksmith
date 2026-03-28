package plan

import (
	"github.com/permanu/docksmith/core"
	"strings"
)

func planBun(fw *core.Framework) (*core.BuildPlan, error) {
	bunImg := ResolveDockerTag("bun", fw.BunVersion)

	buildSteps := []core.Step{
		{Type: core.StepWorkdir, Args: []string{"/app"}},
		{Type: core.StepCopy, Args: []string{"package.json", "bun.lockb*", "bun.lock*", "./"}},
		{Type: core.StepRun, Args: []string{"bun install --frozen-lockfile || bun install"}},
		{Type: core.StepCopy, Args: []string{".", "."}},
	}
	if fw.BuildCommand != "" {
		buildSteps = append(buildSteps, core.Step{
			Type: core.StepRun,
			Args: []string{fw.BuildCommand},
		})
	}

	startParts := strings.Fields(fw.StartCommand)
	if len(startParts) == 0 {
		startParts = []string{"bun", "run", "index.ts"}
	}

	runtimeStage := core.Stage{
		Name: "runtime",
		From: bunImg,
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

	addNonRootUser(&runtimeStage, "bun")
	addHealthcheck(&runtimeStage, "bun", fw.Port)

	return &core.BuildPlan{
		Framework: fw.Name,
		Expose:    fw.Port,
		Stages: []core.Stage{
			{Name: "build", From: bunImg, Steps: buildSteps},
			runtimeStage,
		},
	}, nil
}
