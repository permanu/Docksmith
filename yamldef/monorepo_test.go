// SPDX-License-Identifier: MPL-2.0

package yamldef_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/permanu/docksmith/yamldef"
)

func TestMonorepoFixturePlanResolvesNamedServiceNotRoot(t *testing.T) {
	defs, err := yamldef.LoadFrameworkDefs(frameworksDir(t))
	if err != nil {
		t.Fatalf("load frameworks: %v", err)
	}
	var def *yamldef.FrameworkDef
	for _, candidate := range defs {
		if candidate.Runtime == "go" && candidate.Name == "go" {
			def = candidate
			break
		}
	}
	if def == nil {
		t.Fatal("go framework definition not found")
	}

	var fixture map[string]string
	for _, tc := range def.Tests {
		if strings.Contains(strings.ToLower(tc.Name), "monorepo") {
			fixture = tc.Fixture
			break
		}
	}
	if len(fixture) == 0 {
		t.Fatal("monorepo fixture not found in go-std.yaml")
	}

	dir := t.TempDir()
	for rel, content := range fixture {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	plan, err := yamldef.BuildPlanFromDefDir(def, dir)
	if err != nil {
		t.Fatalf("BuildPlanFromDefDir: %v", err)
	}
	var build string
	for _, stage := range plan.Stages {
		for _, step := range stage.Steps {
			for _, arg := range step.Args {
				if strings.Contains(arg, "go build") {
					build = arg
				}
			}
		}
	}
	if build != "CGO_ENABLED=0 go build -o app ./cmd/deploy" {
		t.Fatalf("build command = %q, want the cmd/deploy service named by the module", build)
	}
}
