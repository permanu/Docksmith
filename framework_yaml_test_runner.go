package docksmith

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// TestResult holds the outcome of a single TestCase run.
type TestResult struct {
	Name   string
	Passed bool
	Reason string
}

// RunFrameworkTests parses the YAML at yamlPath, runs each TestCase against
// the detection engine, and returns one result per case.
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
		results[i] = runTestCase(tc)
	}
	return results, nil
}

// RunFrameworkDefTests runs all inline TestCases from def using the definition's
// own detect rules. Use this when you have a parsed FrameworkDef and want to
// validate it without touching the global detector registry.
func RunFrameworkDefTests(def *FrameworkDef) error {
	for _, tc := range def.Tests {
		r := runTestCaseForDef(def, tc)
		if !r.Passed {
			return fmt.Errorf("[%s] %s", r.Name, r.Reason)
		}
	}
	return nil
}

func runTestCaseForDef(def *FrameworkDef, tc TestCase) TestResult {
	dir, err := os.MkdirTemp("", "docksmith-deftest-*")
	if err != nil {
		return TestResult{Name: tc.Name, Passed: false, Reason: fmt.Sprintf("mktemp: %v", err)}
	}
	defer os.RemoveAll(dir)

	for relPath, content := range tc.Fixture {
		full, pathErr := containedPath(dir, relPath)
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

	fw := evalDefAgainstDir(def, dir)
	detected := fw != nil

	if detected != tc.Expect.Detected {
		return TestResult{
			Name:   tc.Name,
			Passed: false,
			Reason: fmt.Sprintf("detected=%v, want %v (framework=%q)", detected, tc.Expect.Detected, frameworkName(fw)),
		}
	}
	if tc.Expect.Framework != "" && fw != nil && fw.Name != tc.Expect.Framework {
		return TestResult{
			Name:   tc.Name,
			Passed: false,
			Reason: fmt.Sprintf("framework=%q, want %q", fw.Name, tc.Expect.Framework),
		}
	}
	if tc.Expect.Port != 0 && fw != nil && fw.Port != tc.Expect.Port {
		return TestResult{
			Name:   tc.Name,
			Passed: false,
			Reason: fmt.Sprintf("port=%d, want %d", fw.Port, tc.Expect.Port),
		}
	}
	return TestResult{Name: tc.Name, Passed: true}
}

func frameworkName(fw *Framework) string {
	if fw == nil {
		return ""
	}
	return fw.Name
}

func runTestCase(tc TestCase) TestResult {
	dir, err := os.MkdirTemp("", "docksmith-test-*")
	if err != nil {
		return TestResult{Name: tc.Name, Passed: false, Reason: fmt.Sprintf("mktemp: %v", err)}
	}
	defer os.RemoveAll(dir)

	for relPath, content := range tc.Fixture {
		full, pathErr := containedPath(dir, relPath)
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
