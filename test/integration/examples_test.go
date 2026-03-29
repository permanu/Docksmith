package integration_test

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestExamplesCompile(t *testing.T) {
	examples, err := filepath.Glob("../../examples/*/main.go")
	if err != nil {
		t.Fatalf("glob examples: %v", err)
	}
	if len(examples) == 0 {
		t.Fatal("no examples found — expected at least one in ../../examples/*/main.go")
	}

	for _, ex := range examples {
		dir := filepath.Dir(ex)
		name := filepath.Base(dir)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmd := exec.Command("go", "build", "-o", "/dev/null", ".")
			cmd.Dir = dir
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("example %s failed to compile:\n%v\n%s", name, err, out)
			}
		})
	}
}
