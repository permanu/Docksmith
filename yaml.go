package docksmith

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/permanu/docksmith/yamldef"
)

// ---------------------------------------------------------------------------
// YAML framework loader wrappers
// ---------------------------------------------------------------------------

// LoadFrameworkDefs loads YAML framework definitions from dir.
func LoadFrameworkDefs(dir string) ([]*FrameworkDef, error) {
	return yamldef.LoadFrameworkDefs(dir)
}

// LoadAndRegisterFrameworks loads YAML defs from each dir (in order) and
// registers them as detectors. Resolution order after registration:
//  1. .docksmith/frameworks/ in project dir (call with project path first)
//  2. ~/.docksmith/frameworks/ (call with home path second)
//  3. Built-in Go detectors (already registered at init time)
//
// YAML detectors are prepended, so the last dir passed has lowest YAML priority.
// Dirs that don't exist are silently skipped.
func LoadAndRegisterFrameworks(dirs ...string) error {
	var errs []string
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		defs, err := yamldef.LoadFrameworkDefs(dir)
		if err != nil {
			errs = append(errs, err.Error())
		}
		for _, def := range defs {
			d := def // capture
			RegisterDetector("yaml:"+d.Name, func(dir string) *Framework {
				return evalDefAgainstDir(d, dir)
			})
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// evalDefAgainstDir runs the detect rules for def against dir.
func evalDefAgainstDir(def *FrameworkDef, dir string) *Framework {
	name, port, matched := yamldef.EvalDefAgainstDir(def, dir)
	if !matched {
		return nil
	}
	return &Framework{
		Name: name,
		Port: port,
	}
}

// ---------------------------------------------------------------------------
// YAML framework plan builders — thin facades over yamldef
// ---------------------------------------------------------------------------

// BuildPlanFromDef converts a FrameworkDef into a BuildPlan.
// It resolves Base fields via ResolveDockerTag and converts StepDefs to Steps.
func BuildPlanFromDef(def *FrameworkDef, fw *Framework) (*BuildPlan, error) {
	return yamldef.BuildPlanFromDef(def, fw)
}

// BuildPlanFromDefDir builds a BuildPlan from a FrameworkDef by resolving
// version and package manager from the given project directory.
func BuildPlanFromDefDir(def *FrameworkDef, dir string) (*BuildPlan, error) {
	return yamldef.BuildPlanFromDefDir(def, dir)
}

// ---------------------------------------------------------------------------
// YAML framework test runner wrappers
// ---------------------------------------------------------------------------

// RunFrameworkTests delegates to yamldef.RunFrameworkTests.
func RunFrameworkTests(yamlPath string) ([]TestResult, error) {
	return yamldef.RunFrameworkTests(yamlPath)
}

// RunFrameworkDefTests delegates to yamldef.RunFrameworkDefTests.
func RunFrameworkDefTests(def *FrameworkDef) error {
	return yamldef.RunFrameworkDefTests(def)
}

// runTestCase runs a single test case using the global detector registry.
// This function stays in the root package because it calls Detect(), which
// uses the global detector registry — moving it to yamldef would create a
// circular import.
func runTestCase(tc TestCase) TestResult {
	dir, err := os.MkdirTemp("", "docksmith-test-*")
	if err != nil {
		return TestResult{Name: tc.Name, Passed: false, Reason: fmt.Sprintf("mktemp: %v", err)}
	}
	defer os.RemoveAll(dir)

	for relPath, content := range tc.Fixture {
		full, pathErr := yamldef.ContainedPath(dir, relPath)
		if pathErr != nil {
			return TestResult{Name: tc.Name, Passed: false, Reason: fmt.Sprintf("unsafe fixture path %q: %v", relPath, pathErr)}
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return TestResult{Name: tc.Name, Passed: false, Reason: fmt.Sprintf("mkdir %s: %v", relPath, err)}
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			return TestResult{Name: tc.Name, Passed: false, Reason: fmt.Sprintf("write %s: %v", relPath, err)}
		}
	}

	fw, detectErr := Detect(dir)
	if detectErr != nil && !errors.Is(detectErr, ErrNotDetected) {
		return TestResult{Name: tc.Name, Passed: false, Reason: fmt.Sprintf("detect: %v", detectErr)}
	}
	detected := fw != nil && fw.Name != "static"

	if detected != tc.Expect.Detected {
		return TestResult{
			Name:   tc.Name,
			Passed: false,
			Reason: fmt.Sprintf("detected=%v, want %v (framework=%q)", detected, tc.Expect.Detected, fw.Name),
		}
	}

	if tc.Expect.Framework != "" && fw.Name != tc.Expect.Framework {
		return TestResult{
			Name:   tc.Name,
			Passed: false,
			Reason: fmt.Sprintf("framework=%q, want %q", fw.Name, tc.Expect.Framework),
		}
	}

	if tc.Expect.Port != 0 && fw.Port != tc.Expect.Port {
		return TestResult{
			Name:   tc.Name,
			Passed: false,
			Reason: fmt.Sprintf("port=%d, want %d", fw.Port, tc.Expect.Port),
		}
	}

	return TestResult{Name: tc.Name, Passed: true}
}
