package docksmith

import (
	"strings"
	"testing"
)

func TestAddNonRootUser_BuiltIn(t *testing.T) {
	stage := &Stage{Name: "runtime", From: "node:22-alpine"}
	addNonRootUser(stage, "node")

	var userStep *Step
	for i := range stage.Steps {
		if stage.Steps[i].Type == StepUser {
			userStep = &stage.Steps[i]
		}
	}
	if userStep == nil {
		t.Fatal("expected a USER step")
	}
	if userStep.Args[0] != "node" {
		t.Errorf("USER: got %q, want %q", userStep.Args[0], "node")
	}
	// No RUN step for creating a user when builtInUser is non-empty.
	for _, s := range stage.Steps {
		if s.Type == StepRun && strings.Contains(s.Args[0], "addgroup") {
			t.Error("should not add addgroup/adduser RUN when builtInUser is set")
		}
	}
}

func TestAddNonRootUser_Created(t *testing.T) {
	stage := &Stage{Name: "runtime", From: "python:3.12-slim"}
	addNonRootUser(stage, "")

	var runStep *Step
	var userStep *Step
	for i := range stage.Steps {
		s := &stage.Steps[i]
		if s.Type == StepRun && strings.Contains(s.Args[0], "appuser") {
			runStep = s
		}
		if s.Type == StepUser {
			userStep = s
		}
	}
	if runStep == nil {
		t.Error("expected a RUN step creating appuser")
	}
	if userStep == nil {
		t.Fatal("expected a USER step")
	}
	if userStep.Args[0] != "appuser" {
		t.Errorf("USER: got %q, want %q", userStep.Args[0], "appuser")
	}
}

func TestAddNonRootUser_CreatedRunContainsGroupAndUser(t *testing.T) {
	stage := &Stage{Name: "runtime", From: "debian:slim"}
	addNonRootUser(stage, "")

	for _, s := range stage.Steps {
		if s.Type == StepRun && strings.Contains(s.Args[0], "appuser") {
			if !strings.Contains(s.Args[0], "appgroup") {
				t.Error("RUN should create both appgroup and appuser")
			}
			return
		}
	}
	t.Error("no appuser creation step found")
}

func TestAddHealthcheck_Node(t *testing.T) {
	stage := &Stage{Name: "runtime", From: "node:22-alpine"}
	addHealthcheck(stage, "node", 3000)

	hc := findHealthcheck(stage)
	if hc == nil {
		t.Fatal("expected a HEALTHCHECK step")
	}
	if !strings.Contains(hc.Args[0], "3000") {
		t.Errorf("healthcheck should reference port 3000, got: %s", hc.Args[0])
	}
	if !strings.Contains(hc.Args[0], "http.get") && !strings.Contains(hc.Args[0], "http") {
		t.Errorf("node healthcheck should use http, got: %s", hc.Args[0])
	}
}

func TestAddHealthcheck_Python(t *testing.T) {
	stage := &Stage{Name: "runtime", From: "python:3.12-slim"}
	addHealthcheck(stage, "python", 8000)

	hc := findHealthcheck(stage)
	if hc == nil {
		t.Fatal("expected a HEALTHCHECK step")
	}
	if !strings.Contains(hc.Args[0], "8000") {
		t.Errorf("healthcheck should reference port 8000, got: %s", hc.Args[0])
	}
	if !strings.Contains(hc.Args[0], "urllib") {
		t.Errorf("python healthcheck should use urllib, got: %s", hc.Args[0])
	}
}

func TestAddHealthcheck_Go(t *testing.T) {
	// Go uses distroless — no shell, no healthcheck.
	stage := &Stage{Name: "runtime", From: "gcr.io/distroless/static-debian12:nonroot"}
	addHealthcheck(stage, "go", 8080)

	if hc := findHealthcheck(stage); hc != nil {
		t.Error("go runtime should not get a healthcheck (distroless)")
	}
}

func TestAddHealthcheck_Rust(t *testing.T) {
	stage := &Stage{Name: "runtime", From: "gcr.io/distroless/cc-debian12:nonroot"}
	addHealthcheck(stage, "rust", 8080)

	if hc := findHealthcheck(stage); hc != nil {
		t.Error("rust runtime should not get a healthcheck (distroless)")
	}
}

func TestAddHealthcheck_Static(t *testing.T) {
	stage := &Stage{Name: "runtime", From: "nginx:alpine"}
	addHealthcheck(stage, "static", 80)

	hc := findHealthcheck(stage)
	if hc == nil {
		t.Fatal("expected a HEALTHCHECK step for static")
	}
	if !strings.Contains(hc.Args[0], "curl") {
		t.Errorf("static healthcheck should use curl, got: %s", hc.Args[0])
	}
	if !strings.Contains(hc.Args[0], ":80/") {
		t.Errorf("static healthcheck should use port 80, got: %s", hc.Args[0])
	}
}

func TestAddHealthcheck_Bun(t *testing.T) {
	stage := &Stage{Name: "runtime", From: "oven/bun:1"}
	addHealthcheck(stage, "bun", 3000)

	hc := findHealthcheck(stage)
	if hc == nil {
		t.Fatal("expected a HEALTHCHECK step for bun")
	}
	if !strings.Contains(hc.Args[0], "3000") {
		t.Errorf("bun healthcheck should reference port 3000, got: %s", hc.Args[0])
	}
	if !strings.Contains(hc.Args[0], "fetch") {
		t.Errorf("bun healthcheck should use fetch, got: %s", hc.Args[0])
	}
}

func TestAddHealthcheck_Deno(t *testing.T) {
	stage := &Stage{Name: "runtime", From: "denoland/deno:latest"}
	addHealthcheck(stage, "deno", 8000)

	hc := findHealthcheck(stage)
	if hc == nil {
		t.Fatal("expected a HEALTHCHECK step for deno")
	}
	if !strings.Contains(hc.Args[0], "8000") {
		t.Errorf("deno healthcheck should reference port 8000, got: %s", hc.Args[0])
	}
}

func TestAddHealthcheck_Ruby(t *testing.T) {
	stage := &Stage{Name: "runtime", From: "ruby:3.3-slim"}
	addHealthcheck(stage, "ruby", 3000)

	hc := findHealthcheck(stage)
	if hc == nil {
		t.Fatal("expected a HEALTHCHECK step for ruby")
	}
	if !strings.Contains(hc.Args[0], "ruby") {
		t.Errorf("ruby healthcheck should use ruby, got: %s", hc.Args[0])
	}
}

func TestAddHealthcheck_Java(t *testing.T) {
	stage := &Stage{Name: "runtime", From: "eclipse-temurin:21-jre-alpine"}
	addHealthcheck(stage, "java", 8080)

	hc := findHealthcheck(stage)
	if hc == nil {
		t.Fatal("expected a HEALTHCHECK step for java")
	}
	// Alpine JRE images don't have curl — use wget instead.
	if !strings.Contains(hc.Args[0], "wget") {
		t.Errorf("java healthcheck should use wget, got: %s", hc.Args[0])
	}
}

func TestAddTini_CopiesBinary(t *testing.T) {
	builder := &Stage{Name: "builder", From: "node:22-alpine"}
	runtime := &Stage{Name: "runtime", From: "node:22-alpine"}
	addTini(builder, runtime)

	// Builder should install tini via apt.
	var installStep *Step
	for i := range builder.Steps {
		if builder.Steps[i].Type == StepRun && strings.Contains(builder.Steps[i].Args[0], "tini") {
			installStep = &builder.Steps[i]
		}
	}
	if installStep == nil {
		t.Error("builder should install tini")
	}

	// Runtime should COPY tini from builder.
	var copyStep *Step
	for i := range runtime.Steps {
		s := &runtime.Steps[i]
		if s.Type == StepCopyFrom && s.CopyFrom != nil && strings.Contains(s.CopyFrom.Src, "tini") {
			copyStep = s
		}
	}
	if copyStep == nil {
		t.Error("runtime should copy tini from builder")
	}

	// Runtime should have an ENTRYPOINT with tini.
	var epStep *Step
	for i := range runtime.Steps {
		if runtime.Steps[i].Type == StepEntrypoint {
			epStep = &runtime.Steps[i]
		}
	}
	if epStep == nil {
		t.Fatal("runtime should have an ENTRYPOINT for tini")
	}
	joined := strings.Join(epStep.Args, " ")
	if !strings.Contains(joined, "tini") {
		t.Errorf("ENTRYPOINT should reference tini, got: %s", joined)
	}
}

func TestAddTini_AlpineUsesSbinPath(t *testing.T) {
	builder := &Stage{Name: "deps", From: "node:22-alpine"}
	runtime := &Stage{Name: "runtime", From: "node:22-alpine"}
	addTini(builder, runtime)

	for i := range runtime.Steps {
		s := &runtime.Steps[i]
		if s.Type == StepCopyFrom && s.CopyFrom != nil && strings.Contains(s.CopyFrom.Src, "tini") {
			if s.CopyFrom.Src != "/sbin/tini" {
				t.Errorf("alpine tini COPY src: got %q, want /sbin/tini", s.CopyFrom.Src)
			}
			return
		}
	}
	t.Error("no tini COPY step found in runtime")
}

func TestAddTini_DebianUsesUsrBinPath(t *testing.T) {
	builder := &Stage{Name: "deps", From: "node:22-slim"}
	runtime := &Stage{Name: "runtime", From: "node:22-slim"}
	addTini(builder, runtime)

	for i := range runtime.Steps {
		s := &runtime.Steps[i]
		if s.Type == StepCopyFrom && s.CopyFrom != nil && strings.Contains(s.CopyFrom.Src, "tini") {
			if s.CopyFrom.Src != "/usr/bin/tini" {
				t.Errorf("debian tini COPY src: got %q, want /usr/bin/tini", s.CopyFrom.Src)
			}
			return
		}
	}
	t.Error("no tini COPY step found in runtime")
}

func TestWithAptCleanup(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"simple install", "apt-get install -y curl"},
		{"update then install", "apt-get update && apt-get install -y wget"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := withAptCleanup(tt.in)
			if !strings.Contains(out, "rm -rf /var/lib/apt/lists/*") {
				t.Errorf("withAptCleanup(%q) missing cleanup, got: %q", tt.in, out)
			}
			if !strings.HasPrefix(out, tt.in) {
				t.Errorf("withAptCleanup should prepend original cmd, got: %q", out)
			}
		})
	}
}

// findHealthcheck returns the first StepHealthcheck in a stage, or nil.
func findHealthcheck(stage *Stage) *Step {
	for i := range stage.Steps {
		if stage.Steps[i].Type == StepHealthcheck {
			return &stage.Steps[i]
		}
	}
	return nil
}
