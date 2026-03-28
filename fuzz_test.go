package docksmith

import (
	"strings"
	"testing"

	"github.com/permanu/docksmith/config"
)

func FuzzSanitizeDockerfileArg(f *testing.F) {
	f.Add("npm start")
	f.Add("hello\nworld")
	f.Add("")
	f.Fuzz(func(t *testing.T, s string) {
		out := sanitizeDockerfileArg(s)
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
		out := sanitizeAppID(s)
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
		_ = parseVersionString(s)
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
		_, _ = FrameworkFromJSON(data)
	})
}

func FuzzEmitDockerfile(f *testing.F) {
	f.Add("deps", "npm install", "npm start")
	f.Add("build", "go build -o app .", "./app")
	f.Add("", "", "")
	f.Fuzz(func(t *testing.T, stageName, buildCmd, startCmd string) {
		plan := &BuildPlan{
			Framework: "fuzz",
			Expose:    8080,
			Stages: []Stage{{
				Name: stageName,
				From: "alpine:3.19",
				Steps: []Step{
					{Type: StepWorkdir, Args: []string{"/app"}},
					{Type: StepRun, Args: []string{buildCmd}},
					{Type: StepCmd, Args: []string{startCmd}},
					{Type: StepEnv, Args: []string{"KEY", buildCmd}},
					{Type: StepExpose, Args: []string{"8080"}},
				},
			}},
		}
		_ = EmitDockerfile(plan)
	})
}
