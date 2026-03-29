package integration_test

import (
	"strings"
	"testing"

	"github.com/permanu/docksmith"
)

func TestContextRoot_NodeMonorepo_CopyPaths(t *testing.T) {
	fw := &docksmith.Framework{
		Name:           "nextjs",
		BuildCommand:   "npm run build",
		StartCommand:   "npm start",
		Port:           3000,
		NodeVersion:    "22",
		PackageManager: "npm",
	}
	plan, err := docksmith.Plan(fw, docksmith.WithContextRoot("apps/web"))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	out := docksmith.EmitDockerfile(plan)

	assertContains(t, out, "COPY ./apps/web/package.json")
	assertContains(t, out, "COPY ./apps/web .")

	// Should NOT contain bare "COPY . ." (without prefix).
	lines := strings.Split(out, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "COPY . ." || trimmed == "COPY --link . ." {
			t.Errorf("bare 'COPY . .' found — should be prefixed with apps/web:\n%s", out)
		}
	}
}

func TestContextRoot_GoMonorepo_CopyPaths(t *testing.T) {
	fw := &docksmith.Framework{
		Name:      "go",
		GoVersion: "1.26",
		Port:      8080,
	}
	plan, err := docksmith.Plan(fw, docksmith.WithContextRoot("services/api"))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	out := docksmith.EmitDockerfile(plan)

	assertContains(t, out, "COPY ./services/api/go.mod")
	assertContains(t, out, "COPY ./services/api .")
}

func TestContextRoot_PythonMonorepo_CopyPaths(t *testing.T) {
	fw := &docksmith.Framework{
		Name:          "django",
		PythonVersion: "3.12",
		PythonPM:      "pip",
		Port:          8000,
		StartCommand:  "gunicorn myapp.wsgi:application",
	}
	plan, err := docksmith.Plan(fw, docksmith.WithContextRoot("apps/backend"))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	out := docksmith.EmitDockerfile(plan)

	assertContains(t, out, "COPY ./apps/backend/requirements.txt")
	assertContains(t, out, "COPY ./apps/backend .")
}

func TestContextRoot_NotSet_BackwardCompat(t *testing.T) {
	fw := &docksmith.Framework{
		Name:           "nextjs",
		BuildCommand:   "npm run build",
		StartCommand:   "npm start",
		Port:           3000,
		NodeVersion:    "22",
		PackageManager: "npm",
	}
	plan, err := docksmith.Plan(fw)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	out := docksmith.EmitDockerfile(plan)

	// Without context root, should have bare COPY . .
	assertContains(t, out, "COPY . .")
	if strings.Contains(out, "COPY ./") {
		t.Errorf("without context root, no ./ prefix expected:\n%s", out)
	}
}

func TestValidateContextRoot_Integration(t *testing.T) {
	tests := []struct {
		name        string
		contextRoot string
		appDir      string
		wantRel     string
		wantErr     bool
	}{
		{"valid monorepo", "/repo", "/repo/apps/web", "apps/web", false},
		{"same dir", "/repo", "/repo", "", false},
		{"not ancestor", "/repo", "/other/app", "", true},
		{"traversal in root", "/repo/../etc", "/repo/app", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rel, err := docksmith.ValidateContextRoot(tt.contextRoot, tt.appDir)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rel != tt.wantRel {
				t.Errorf("rel = %q, want %q", rel, tt.wantRel)
			}
		})
	}
}
