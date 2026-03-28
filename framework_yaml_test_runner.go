package docksmith

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/permanu/docksmith/yamldef"
)

// RunFrameworkTests delegates to yamldef.RunFrameworkTests.
func RunFrameworkTests(yamlPath string) ([]TestResult, error) {
	return yamldef.RunFrameworkTests(yamlPath)
}

// RunFrameworkDefTests delegates to yamldef.RunFrameworkDefTests.
func RunFrameworkDefTests(def *FrameworkDef) error {
	return yamldef.RunFrameworkDefTests(def)
}

// runTestCaseForDef runs a single test case against the definition's own
// detection rules. Delegates to yamldef.RunTestCaseForDef.
func runTestCaseForDef(def *FrameworkDef, tc TestCase) TestResult {
	return yamldef.RunTestCaseForDef(def, tc)
}

func frameworkName(fw *Framework) string {
	if fw == nil {
		return ""
	}
	return fw.Name
}

// runTestCase runs a single test case using the global detector registry.
// This remains in root because it calls root's Detect function.
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
