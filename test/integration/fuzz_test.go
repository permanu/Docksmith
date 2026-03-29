package integration_test

import (
	"strings"
	"testing"

	"github.com/permanu/docksmith"
	"github.com/permanu/docksmith/config"
	"github.com/permanu/docksmith/detect"
	"github.com/permanu/docksmith/emit"
	"github.com/permanu/docksmith/plan"
)

func FuzzSanitizeDockerfileArg(f *testing.F) {
	f.Add("npm start")
	f.Add("hello\nworld")
	f.Add("")
	f.Fuzz(func(t *testing.T, s string) {
		out := emit.SanitizeDockerfileArg(s)
		if strings.ContainsAny(out, "\n\r\x00") {
			t.Fatalf("output contains forbidden char: %q", out)
		}
	})
}

func FuzzSanitizeAppID(f *testing.F) {
	f.Add("my-app")
	f.Add("../../../etc")
	f.Add("....")
	f.Add("")
	f.Fuzz(func(t *testing.T, s string) {
		out := plan.SanitizeAppID(s)
		if strings.Contains(out, "..") {
			t.Fatalf("output contains '..': %q", out)
		}
		if strings.ContainsAny(out, "/\\\x00") {
			t.Fatalf("output contains forbidden char: %q", out)
		}
		if out == "" {
			t.Fatal("output is empty")
		}
	})
}

func FuzzParseVersionString(f *testing.F) {
	f.Add("18.0.0")
	f.Add(">=3.9,<4")
	f.Add("lts/*")
	f.Add("")
	f.Add("v20.1.0")
	f.Fuzz(func(t *testing.T, s string) {
		_ = detect.ParseVersionString(s)
	})
}

func FuzzParseConfig(f *testing.F) {
	f.Add([]byte("runtime = \"node\"\nstart = \"npm start\"\n"))
	f.Add([]byte{})
	f.Add([]byte("{{invalid"))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = config.ParseConfig("test.toml", data)
		_, _ = config.ParseConfig("test.yaml", data)
		_, _ = config.ParseConfig("test.json", data)
	})
}

func FuzzFrameworkFromJSON(f *testing.F) {
	f.Add([]byte(`{"name":"nextjs"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = docksmith.FrameworkFromJSON(data)
	})
}

func FuzzEmitDockerfile(f *testing.F) {
	f.Add("deps", "npm install", "npm start")
	f.Add("build", "go build -o app .", "./app")
	f.Add("", "", "")
	f.Fuzz(func(t *testing.T, stageName, buildCmd, startCmd string) {
		p := &docksmith.BuildPlan{
			Framework: "fuzz",
			Expose:    8080,
			Stages: []docksmith.Stage{{
				Name: stageName,
				From: "alpine:3.19",
				Steps: []docksmith.Step{
					{Type: docksmith.StepWorkdir, Args: []string{"/app"}},
					{Type: docksmith.StepRun, Args: []string{buildCmd}},
					{Type: docksmith.StepCmd, Args: []string{startCmd}},
					{Type: docksmith.StepEnv, Args: []string{"KEY", buildCmd}},
					{Type: docksmith.StepExpose, Args: []string{"8080"}},
				},
			}},
		}
		_ = docksmith.EmitDockerfile(p)
	})
}
