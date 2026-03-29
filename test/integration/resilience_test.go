package integration_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/permanu/docksmith"
	"github.com/permanu/docksmith/config"
	"github.com/permanu/docksmith/yamldef"
)

func TestParseConfig_malformed(t *testing.T) {
	bom := "\xef\xbb\xbf"
	longRuntime := strings.Repeat("x", 10000)

	cases := []struct {
		name    string
		file    string
		data    []byte
		wantErr bool
	}{
		{"empty yaml", "c.yaml", nil, false},
		{"empty json", "c.json", nil, true},
		{"empty toml", "c.toml", nil, false},
		{"binary yaml", "c.yaml", []byte{0xff, 0xfe, 0x00, 0x01, 0x80}, true},
		{"binary json", "c.json", []byte{0xff, 0xfe, 0x00, 0x01, 0x80}, true},
		{"binary toml", "c.toml", []byte{0xff, 0xfe, 0x00, 0x01, 0x80}, true},
		{"truncated json", "c.json", []byte(`{"runtime": "node"`), true},
		{"wrong type json", "c.json", []byte(`{"runtime_config": {"expose": "not-a-number"}}`), true},
		{"wrong type yaml", "c.yaml", []byte("runtime_config:\n  expose: not-a-number\n"), true},
		{"bom yaml", "c.yaml", []byte(bom + `runtime: node`), false},
		{"long runtime json", "c.json", []byte(`{"runtime":"` + longRuntime + `","start":{"command":"x"}}`), false},
		{"negative port json", "c.json", []byte(`{"runtime":"node","start":{"command":"x"},"runtime_config":{"expose":-1}}`), false},
		{"unknown fields json", "c.json", []byte(`{"runtime":"node","start":{"command":"x"},"flavor":"vanilla"}`), false},
		{"unknown fields yaml", "c.yaml", []byte("runtime: node\nstart:\n  command: x\nflavor: vanilla\n"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := config.ParseConfig(tc.file, tc.data)
			if tc.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			_ = cfg
		})
	}
}

func TestParseConfig_validation(t *testing.T) {
	cases := []struct {
		name    string
		file    string
		data    []byte
		wantErr bool
	}{
		{"long runtime fails validation", "c.json",
			[]byte(`{"runtime":"` + strings.Repeat("x", 10000) + `","start":{"command":"x"}}`), true},
		{"negative port still parses", "c.json",
			[]byte(`{"runtime":"node","start":{"command":"x"},"runtime_config":{"expose":-1}}`), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := config.ParseConfig(tc.file, tc.data)
			if err != nil {
				return
			}
			err = cfg.Validate()
			if tc.wantErr && err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestFrameworkFromJSON_malformed(t *testing.T) {
	cases := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{"empty", nil, true},
		{"null", []byte("null"), false},
		{"empty object", []byte("{}"), false},
		{"binary", []byte{0x00, 0xff, 0xfe}, true},
		{"array", []byte("[1,2,3]"), true},
		{"truncated", []byte(`{"name":"next`), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fw, err := docksmith.FrameworkFromJSON(tc.data)
			if tc.wantErr && err == nil {
				t.Error("expected error")
			}
			_ = fw
		})
	}
}

func TestEvalDetectRules_adversarial(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "app.js"), []byte("console.log('hi')"), 0644)

	manyRules := make([]docksmith.DetectRule, 1000)
	for i := range manyRules {
		manyRules[i] = docksmith.DetectRule{File: "app.js"}
	}

	cases := []struct {
		name  string
		rules docksmith.DetectRules
		want  bool
	}{
		{"empty rules", docksmith.DetectRules{}, true},
		{"1000 all rules", docksmith.DetectRules{All: manyRules}, true},
		{"empty file+dir rule", docksmith.DetectRules{All: []docksmith.DetectRule{{}}}, true},
		{"redos pattern", docksmith.DetectRules{All: []docksmith.DetectRule{
			{File: "app.js", Regex: `(a+)+$`},
		}}, false},
		{"long contains", docksmith.DetectRules{All: []docksmith.DetectRule{
			{File: "app.js", Contains: strings.Repeat("z", 10000)},
		}}, false},
		{"file pointing to dir", docksmith.DetectRules{All: []docksmith.DetectRule{
			{File: "."},
		}}, true},
		{"regex over max len", docksmith.DetectRules{All: []docksmith.DetectRule{
			{File: "app.js", Regex: strings.Repeat("a", 2000)},
		}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := yamldef.EvalDetectRules(dir, tc.rules)
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEvalDetectRules_redosTimeout(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "evil.txt"), []byte(strings.Repeat("a", 50)+"b"), 0644)

	rules := docksmith.DetectRules{All: []docksmith.DetectRule{
		{File: "evil.txt", Regex: `(a+)+$`},
	}}
	_ = yamldef.EvalDetectRules(dir, rules)
}

func TestLoadFrameworkDefs_adversarial(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "good.yaml"), []byte("name: testfw\nruntime: node\n"), 0644)
	os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte{0xff, 0xfe, 0x00}, 0644)
	os.WriteFile(filepath.Join(dir, "cycle.yaml"), []byte("name: cyc\na: &a\n  b: *a\n"), 0644)
	os.WriteFile(filepath.Join(dir, "huge.yaml"), []byte("name: huge\ndata: "+strings.Repeat("x", 100_000)+"\n"), 0644)
	os.WriteFile(filepath.Join(dir, "skip.txt"), []byte("should be ignored"), 0644)
	os.WriteFile(filepath.Join(dir, "noname.yaml"), []byte("runtime: python\n"), 0644)

	defs, err := docksmith.LoadFrameworkDefs(dir)
	if err == nil {
		t.Error("expected partial error for bad files")
	}

	names := map[string]bool{}
	for _, d := range defs {
		names[d.Name] = true
	}
	if !names["testfw"] {
		t.Error("good.yaml should load")
	}
	if !names["huge"] {
		t.Error("huge.yaml should load")
	}
}

func TestLoadFrameworkDefs_nonexistentDir(t *testing.T) {
	_, err := docksmith.LoadFrameworkDefs("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for nonexistent dir")
	}
}

func TestBuildPlan_adversarial(t *testing.T) {
	t.Run("zero stages validates error", func(t *testing.T) {
		p := &docksmith.BuildPlan{Framework: "test", Stages: nil, Expose: 3000}
		if err := p.Validate(); err == nil {
			t.Error("expected error for 0 stages")
		}
	})

	t.Run("100 stages", func(t *testing.T) {
		stages := make([]docksmith.Stage, 100)
		for i := range stages {
			stages[i] = docksmith.Stage{
				From:  "node:20",
				Steps: []docksmith.Step{{Type: docksmith.StepRun, Args: []string{"echo hi"}}},
			}
		}
		p := &docksmith.BuildPlan{Framework: "test", Stages: stages, Expose: 3000}
		if err := p.Validate(); err != nil {
			t.Errorf("100 stages should be fine: %v", err)
		}
	})

	t.Run("stage with 0 steps", func(t *testing.T) {
		p := &docksmith.BuildPlan{
			Framework: "test",
			Stages:    []docksmith.Stage{{Name: "empty", From: "node:20", Steps: nil}},
			Expose:    3000,
		}
		if err := p.Validate(); err == nil {
			t.Error("expected error for empty stage")
		}
	})

	t.Run("unknown from reference", func(t *testing.T) {
		p := &docksmith.BuildPlan{
			Framework: "test",
			Stages: []docksmith.Stage{{
				Name:  "run",
				From:  "nonexistent",
				Steps: []docksmith.Step{{Type: docksmith.StepRun, Args: []string{"echo"}}},
			}},
			Expose: 3000,
		}
		if err := p.Validate(); err == nil {
			t.Error("expected error for unknown from")
		}
	})
}

func TestEmitDockerfile_edgeCases(t *testing.T) {
	t.Run("empty plan", func(t *testing.T) {
		p := &docksmith.BuildPlan{Stages: nil}
		out := docksmith.EmitDockerfile(p)
		if out != "" {
			t.Error("expected empty output for empty plan")
		}
	})

	t.Run("nil steps in stage", func(t *testing.T) {
		p := &docksmith.BuildPlan{
			Stages: []docksmith.Stage{{Name: "base", From: "alpine:3", Steps: nil}},
			Expose: 8080,
		}
		out := docksmith.EmitDockerfile(p)
		if !strings.Contains(out, "FROM") {
			t.Error("should still emit FROM")
		}
	})

	t.Run("nil copy_from pointer handled", func(t *testing.T) {
		p := &docksmith.BuildPlan{
			Stages: []docksmith.Stage{{
				Name: "base",
				From: "alpine:3",
				Steps: []docksmith.Step{
					{Type: docksmith.StepCopyFrom, CopyFrom: nil},
				},
			}},
		}
		out := docksmith.EmitDockerfile(p)
		if !strings.Contains(out, "FROM") {
			t.Error("should still emit FROM")
		}
	})

	t.Run("step with nil args", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panicked on nil args: %v", r)
			}
		}()
		p := &docksmith.BuildPlan{
			Stages: []docksmith.Stage{{
				Name:  "base",
				From:  "alpine:3",
				Steps: []docksmith.Step{{Type: docksmith.StepEnv, Args: nil}},
			}},
		}
		_ = docksmith.EmitDockerfile(p)
	})
}

func TestDetect_emptyAndBrokenDirs(t *testing.T) {
	t.Run("nonexistent dir", func(t *testing.T) {
		fw, err := docksmith.Detect("/nonexistent/dir/xyz")
		_ = fw
		_ = err
	})

	t.Run("file not dir", func(t *testing.T) {
		f, _ := os.CreateTemp("", "docksmith-test-*")
		f.Close()
		defer os.Remove(f.Name())
		fw, err := docksmith.Detect(f.Name())
		_ = fw
		_ = err
	})

	t.Run("empty dir returns ErrNotDetected", func(t *testing.T) {
		dir := t.TempDir()
		_, err := docksmith.Detect(dir)
		if err == nil {
			t.Fatal("expected error for empty dir")
		}
		if !errors.Is(err, docksmith.ErrNotDetected) {
			t.Errorf("error = %v, want ErrNotDetected", err)
		}
	})

	t.Run("unreadable dir", func(t *testing.T) {
		dir := t.TempDir()
		sub := filepath.Join(dir, "noperm")
		os.Mkdir(sub, 0000)
		defer os.Chmod(sub, 0755)
		fw, err := docksmith.Detect(sub)
		_ = fw
		_ = err
	})
}

func TestBuildPlanFromDef_nilAndEmpty(t *testing.T) {
	t.Run("nil def", func(t *testing.T) {
		_, err := docksmith.BuildPlanFromDefDir(nil, t.TempDir())
		if err == nil {
			t.Error("expected error for nil def")
		}
	})

	t.Run("def with no stages", func(t *testing.T) {
		def := &docksmith.FrameworkDef{Name: "empty", Runtime: "node"}
		p, err := docksmith.BuildPlanFromDefDir(def, t.TempDir())
		if err != nil && p != nil {
			if len(p.Stages) != 0 {
				t.Error("expected 0 stages")
			}
		}
	})

	t.Run("stage with no base or from", func(t *testing.T) {
		def := &docksmith.FrameworkDef{
			Name:    "broken",
			Runtime: "node",
			Plan: docksmith.PlanDef{
				Port:   3000,
				Stages: []docksmith.StageDef{{Name: "oops", Steps: []docksmith.StepDef{{Run: "echo"}}}},
			},
		}
		_, err := docksmith.BuildPlanFromDefDir(def, t.TempDir())
		if err == nil {
			t.Error("expected error for stage with no base/from")
		}
	})
}
