package yamldef

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// RunFrameworkTests parses the YAML at yamlPath, runs each TestCase against
// the definition's own detection rules, and returns one result per case.
func RunFrameworkTests(yamlPath string) ([]TestResult, error) {
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", yamlPath, err)
	}

	var def FrameworkDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parse %s: %w", yamlPath, err)
	}

	if len(def.Tests) == 0 {
		return nil, fmt.Errorf("%s: no tests defined", yamlPath)
	}

	results := make([]TestResult, len(def.Tests))
	for i, tc := range def.Tests {
		results[i] = RunTestCaseForDef(&def, tc)
	}
	return results, nil
}

// RunFrameworkDefTests runs all inline TestCases from def using the definition's
// own detect rules. Use this when you have a parsed FrameworkDef and want to
// validate it without touching the global detector registry.
func RunFrameworkDefTests(def *FrameworkDef) error {
	for _, tc := range def.Tests {
		r := RunTestCaseForDef(def, tc)
		if !r.Passed {
			return fmt.Errorf("[%s] %s", r.Name, r.Reason)
		}
	}
	return nil
}

// EvalDefAgainstDir runs the detect rules for def against dir.
// Returns the framework name and port on match, or empty name on no match.
func EvalDefAgainstDir(def *FrameworkDef, dir string) (name string, port int, matched bool) {
	if !EvalDetectRules(dir, def.Detect) {
		return "", 0, false
	}
	port = def.Plan.Port
	return def.Name, port, true
}

// RunTestCaseForDef runs a single test case against the given definition's
// detect rules and returns the result.
func RunTestCaseForDef(def *FrameworkDef, tc TestCase) TestResult {
	dir, err := os.MkdirTemp("", "docksmith-deftest-*")
	if err != nil {
		return TestResult{Name: tc.Name, Passed: false, Reason: fmt.Sprintf("mktemp: %v", err)}
	}
	defer os.RemoveAll(dir)

	for relPath, content := range tc.Fixture {
		full, pathErr := ContainedPath(dir, relPath)
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

	name, port, detected := EvalDefAgainstDir(def, dir)

	if detected != tc.Expect.Detected {
		return TestResult{
			Name:   tc.Name,
			Passed: false,
			Reason: fmt.Sprintf("detected=%v, want %v (framework=%q)", detected, tc.Expect.Detected, name),
		}
	}
	if tc.Expect.Framework != "" && detected && name != tc.Expect.Framework {
		return TestResult{
			Name:   tc.Name,
			Passed: false,
			Reason: fmt.Sprintf("framework=%q, want %q", name, tc.Expect.Framework),
		}
	}
	if tc.Expect.Port != 0 && detected && port != tc.Expect.Port {
		return TestResult{
			Name:   tc.Name,
			Passed: false,
			Reason: fmt.Sprintf("port=%d, want %d", port, tc.Expect.Port),
		}
	}
	return TestResult{Name: tc.Name, Passed: true}
}
