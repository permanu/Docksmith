package docksmith

import (
	"regexp"

	"github.com/permanu/docksmith/yamldef"
)

// evalDetectRules delegates to yamldef.EvalDetectRules.
func evalDetectRules(dir string, rules DetectRules) bool {
	return yamldef.EvalDetectRules(dir, rules)
}

// evalRule delegates to yamldef.EvalRule.
func evalRule(dir string, rule DetectRule) bool {
	return yamldef.EvalRule(dir, rule)
}

// isYAMLFile delegates to yamldef.IsYAMLFile.
func isYAMLFile(name string) bool {
	return yamldef.IsYAMLFile(name)
}

// extractDotPath delegates to yamldef.ExtractDotPath.
func extractDotPath(root any, dotPath string) any {
	return yamldef.ExtractDotPath(root, dotPath)
}

// fileMatchesRegex delegates to yamldef for regex matching against file content.
func fileMatchesRegex(path, pattern string) bool {
	if len(pattern) > 1024 {
		return false
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	data, err := yamldef.ReadFileLimited(path)
	if err != nil {
		return false
	}
	return re.Match(data)
}
