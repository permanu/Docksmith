package yamldef_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/permanu/docksmith/yamldef"
)

func frameworksDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// frameworks/ is in the parent (root) directory
	return filepath.Join(filepath.Dir(file), "..", "frameworks")
}

func TestRunFrameworkTests_nextjs(t *testing.T) {
	defs, err := yamldef.LoadFrameworkDefs(frameworksDir(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	var nextjsDef *yamldef.FrameworkDef
	for _, d := range defs {
		if d.Name == "nextjs" {
			nextjsDef = d
			break
		}
	}
	if nextjsDef == nil {
		t.Fatal("nextjs.yaml not found in frameworks/")
	}

	results, err := yamldef.RunFrameworkTests(filepath.Join(frameworksDir(t), "nextjs.yaml"))
	if err != nil {
		t.Fatalf("RunFrameworkTests: %v", err)
	}
	for _, r := range results {
		if !r.Passed {
			t.Errorf("test %q failed: %s", r.Name, r.Reason)
		}
	}
}

func TestRunFrameworkTests_django(t *testing.T) {
	results, err := yamldef.RunFrameworkTests(filepath.Join(frameworksDir(t), "django.yaml"))
	if err != nil {
		t.Fatalf("RunFrameworkTests: %v", err)
	}
	for _, r := range results {
		if !r.Passed {
			t.Errorf("test %q failed: %s", r.Name, r.Reason)
		}
	}
}

func TestRunFrameworkTests_goStd(t *testing.T) {
	results, err := yamldef.RunFrameworkTests(filepath.Join(frameworksDir(t), "go-std.yaml"))
	if err != nil {
		t.Fatalf("RunFrameworkTests: %v", err)
	}
	for _, r := range results {
		if !r.Passed {
			t.Errorf("test %q failed: %s", r.Name, r.Reason)
		}
	}
}

func TestRunFrameworkDefTests_inline(t *testing.T) {
	def := &yamldef.FrameworkDef{
		Name: "mock-fw",
		Plan: yamldef.PlanDef{Port: 9999},
		Detect: yamldef.DetectRules{
			All: []yamldef.DetectRule{
				{File: "mockfw.config.js"},
			},
		},
		Tests: []yamldef.TestCase{
			{
				Name:    "config file present",
				Fixture: map[string]string{"mockfw.config.js": "module.exports = {}"},
				Expect:  yamldef.TestExpect{Detected: true, Framework: "mock-fw", Port: 9999},
			},
			{
				Name:    "empty dir",
				Fixture: map[string]string{},
				Expect:  yamldef.TestExpect{Detected: false},
			},
		},
	}

	if err := yamldef.RunFrameworkDefTests(def); err != nil {
		t.Fatalf("RunFrameworkDefTests: %v", err)
	}
}
