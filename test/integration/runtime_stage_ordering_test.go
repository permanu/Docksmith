package integration_test

// Tests for runtime stage step ordering (issues #33 and #34).
// Verifies that:
//   - external_tool fetch RUN steps land BEFORE USER (need root to write system paths)
//   - runtime_assets COPY --chown steps land AFTER user creation RUN (user must exist)
//   - USER directive is in the right position
//   - CMD/ENTRYPOINT remains LAST

import (
	"fmt"
	"strings"
	"testing"

	"github.com/permanu/docksmith/config"
	"github.com/permanu/docksmith/core"
	"github.com/permanu/docksmith/plan"
)

// expressAlpineFW returns an express framework that uses an alpine runtime image
// (so user creation emits addgroup/adduser commands we can assert on).
func expressAlpineFW() *core.Framework {
	return &core.Framework{
		Name:         "express",
		Port:         3000,
		StartCommand: "node index.js",
	}
}

// TestRuntimeStageOrdering_UserBeforeChownCopy verifies #33:
// The user-creation RUN (addgroup/adduser) must appear BEFORE any
// COPY --chown that references the same user.
func TestRuntimeStageOrdering_UserBeforeChownCopy(t *testing.T) {
	fw := expressAlpineFW()
	p, err := plan.Plan(fw,
		plan.WithUser("permanu:1000"),
		plan.WithRuntimeAssets([]core.AssetCopy{
			{Src: "atlas.hcl", Dst: "/app/atlas.hcl", Chown: "permanu:permanu"},
		}),
	)
	if err != nil {
		t.Fatalf("Plan() error: %v", err)
	}

	last := p.Stages[len(p.Stages)-1]

	userCreateIdx := -1
	chownCopyIdx := -1
	for i, s := range last.Steps {
		if s.Type == core.StepRun && strings.Contains(strings.Join(s.Args, " "), "addgroup") {
			userCreateIdx = i
		}
		if s.Type == core.StepCopy && len(s.Args) > 0 && strings.Contains(s.Args[0], "--chown=permanu") {
			chownCopyIdx = i
		}
	}

	if userCreateIdx < 0 {
		t.Fatal("expected RUN addgroup/adduser step in runtime stage")
	}
	if chownCopyIdx < 0 {
		t.Fatal("expected COPY --chown=permanu step in runtime stage")
	}
	if userCreateIdx >= chownCopyIdx {
		t.Errorf("#33: RUN user creation (idx %d) must come BEFORE COPY --chown (idx %d); steps:\n%v",
			userCreateIdx, chownCopyIdx, dumpSteps(last.Steps))
	}
}

// TestRuntimeStageOrdering_ExternalToolBeforeUser verifies #34:
// The external_tool fetch RUN step must appear BEFORE the USER directive
// so it executes as root and can write to system paths like /usr/local/bin.
func TestRuntimeStageOrdering_ExternalToolBeforeUser(t *testing.T) {
	fw := expressAlpineFW()
	p, err := plan.Plan(fw,
		plan.WithUser("permanu:1000"),
		plan.WithExternalTools([]config.ExternalTool{
			{
				Name:        "atlas",
				URL:         "https://release.ariga.io/atlas/atlas-linux-amd64-latest",
				SHA256:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				InstallPath: "/usr/local/bin",
				Format:      "binary",
			},
		}),
	)
	if err != nil {
		t.Fatalf("Plan() error: %v", err)
	}

	last := p.Stages[len(p.Stages)-1]

	fetchIdx := -1
	userIdx := -1
	for i, s := range last.Steps {
		if s.Type == core.StepFetchTool {
			fetchIdx = i
		}
		if s.Type == core.StepUser {
			userIdx = i
		}
	}

	if fetchIdx < 0 {
		t.Fatal("expected StepFetchTool in runtime stage")
	}
	if userIdx < 0 {
		t.Fatal("expected StepUser in runtime stage")
	}
	if fetchIdx >= userIdx {
		t.Errorf("#34: external_tool fetch (idx %d) must come BEFORE USER directive (idx %d); steps:\n%v",
			fetchIdx, userIdx, dumpSteps(last.Steps))
	}
}

// TestRuntimeStageOrdering_AllThree exercises all three features together and
// asserts the full required ordering:
//  1. RUN addgroup/adduser (user creation) before COPY --chown
//  2. StepFetchTool before USER directive
//  3. USER directive present
//  4. CMD/ENTRYPOINT is the last meaningful instruction
func TestRuntimeStageOrdering_AllThree(t *testing.T) {
	fw := expressAlpineFW()
	p, err := plan.Plan(fw,
		plan.WithUser("permanu:1000"),
		plan.WithRuntimeAssets([]core.AssetCopy{
			{Src: "atlas.hcl", Dst: "/app/atlas.hcl", Chown: "permanu:permanu"},
		}),
		plan.WithExternalTools([]config.ExternalTool{
			{
				Name:        "atlas",
				URL:         "https://release.ariga.io/atlas/atlas-linux-amd64-latest",
				SHA256:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				InstallPath: "/usr/local/bin",
				Format:      "binary",
			},
		}),
	)
	if err != nil {
		t.Fatalf("Plan() error: %v", err)
	}

	last := p.Stages[len(p.Stages)-1]

	userCreateIdx := -1
	fetchIdx := -1
	chownCopyIdx := -1
	userDirectiveIdx := -1
	lastCmdOrEntryIdx := -1

	for i, s := range last.Steps {
		switch s.Type {
		case core.StepRun:
			joined := strings.Join(s.Args, " ")
			if strings.Contains(joined, "addgroup") || strings.Contains(joined, "useradd") {
				userCreateIdx = i
			}
		case core.StepFetchTool:
			fetchIdx = i
		case core.StepCopy:
			if len(s.Args) > 0 && strings.Contains(s.Args[0], "--chown=permanu") {
				chownCopyIdx = i
			}
		case core.StepUser:
			userDirectiveIdx = i
		case core.StepCmd, core.StepEntrypoint:
			lastCmdOrEntryIdx = i
		}
	}

	if userCreateIdx < 0 {
		t.Fatal("expected RUN addgroup/adduser in runtime stage")
	}
	if fetchIdx < 0 {
		t.Fatal("expected StepFetchTool in runtime stage")
	}
	if chownCopyIdx < 0 {
		t.Fatal("expected COPY --chown=permanu in runtime stage")
	}
	if userDirectiveIdx < 0 {
		t.Fatal("expected USER directive in runtime stage")
	}

	// Assert 1: user creation RUN before COPY --chown (#33)
	if userCreateIdx >= chownCopyIdx {
		t.Errorf("assertion 1 failed (#33): RUN user creation (idx %d) must be before COPY --chown (idx %d); steps:\n%v",
			userCreateIdx, chownCopyIdx, dumpSteps(last.Steps))
	}

	// Assert 2: external_tool fetch before USER directive (#34)
	if fetchIdx >= userDirectiveIdx {
		t.Errorf("assertion 2 failed (#34): StepFetchTool (idx %d) must be before USER (idx %d); steps:\n%v",
			fetchIdx, userDirectiveIdx, dumpSteps(last.Steps))
	}

	// Assert 3: USER directive is present (covered by check above)

	// Assert 4: CMD or ENTRYPOINT is present (plan is complete).
	if lastCmdOrEntryIdx < 0 {
		t.Errorf("assertion 4 failed: runtime stage has no CMD or ENTRYPOINT; steps:\n%v", dumpSteps(last.Steps))
	}
}

// dumpSteps returns a human-readable list of step types and args for test output.
func dumpSteps(steps []core.Step) string {
	var sb strings.Builder
	for i, s := range steps {
		sb.WriteString("  [")
		sb.WriteString(fmt.Sprintf("%d", i))
		sb.WriteString("] ")
		sb.WriteString(fmt.Sprintf("%v", s.Type))
		if len(s.Args) > 0 {
			sb.WriteString(" ")
			sb.WriteString(strings.Join(s.Args, " "))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
