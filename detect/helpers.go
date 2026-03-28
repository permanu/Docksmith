package detect

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxFileReadBytes = 10 << 20 // 10 MB

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func hasFile(dir, name string) bool {
	if name == "" {
		return false
	}
	return fileExists(filepath.Join(dir, name))
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func fileContains(path, substr string) bool {
	data, err := readFileLimited(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), substr)
}

// parseVersionString cleans a raw version string from .nvmrc, .node-version,
// or semver constraint fields. Handles ranges like ">=3.9,<4" by taking the
// first constraint. Returns "" for aliases (lts/*, stable, node).
func parseVersionString(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimSpace(s)
	if s == "" || s == "lts/*" || s == "stable" || s == "node" {
		return ""
	}
	hasComma := strings.Contains(s, ",")
	hasOperator := len(s) > 0 && (s[0] == '>' || s[0] == '<' || s[0] == '=' || s[0] == '~' || s[0] == '^')
	if !hasComma && !hasOperator {
		// Plain version from .nvmrc/.node-version — return as-is.
		return s
	}
	if hasComma {
		s = strings.TrimSpace(s[:strings.Index(s, ",")])
	}
	s = strings.TrimLeft(s, "><=~^")
	s = strings.TrimSpace(s)
	parts := strings.SplitN(s, ".", 3)
	major := parts[0]
	if major == "" || !isDigits(major) {
		return ""
	}
	if len(parts) < 2 {
		return major
	}
	minor := strings.TrimSuffix(parts[1], "x")
	if minor == "" || !isDigits(minor) || minor == "0" {
		return major
	}
	return major + "." + minor
}

// extractMajorVersion pulls a usable version from semver constraints, keeping
// major.minor when minor is meaningful. "3.9.1" -> "3.9", "18.0.0" -> "18".
func extractMajorVersion(constraint string) string {
	constraint = strings.TrimSpace(constraint)
	if constraint == "" || constraint == "*" {
		return ""
	}
	constraint = strings.TrimLeft(constraint, "><=~^")
	constraint = strings.TrimSpace(constraint)
	parts := strings.SplitN(constraint, ".", 3)
	major := strings.TrimSpace(parts[0])
	if major == "" || !isDigits(major) {
		return ""
	}
	if len(parts) < 2 {
		return major
	}
	minor := strings.TrimSuffix(strings.TrimSpace(parts[1]), "x")
	if minor == "" || !isDigits(minor) || minor == "0" {
		return major
	}
	return major + "." + minor
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// readFileLimited reads up to maxFileReadBytes from path.
// Returns an error if the file exceeds the limit.
func readFileLimited(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	lr := io.LimitReader(f, maxFileReadBytes+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxFileReadBytes {
		return nil, fmt.Errorf("file %s exceeds %d byte limit", filepath.Base(path), maxFileReadBytes)
	}
	return data, nil
}

// containedPath joins base and rel, then verifies the result is under base.
// Prevents path traversal via "../" or absolute paths in rel.
func containedPath(base, rel string) (string, error) {
	if rel == "" {
		return "", fmt.Errorf("empty path")
	}
	// Strip null bytes.
	rel = strings.ReplaceAll(rel, "\x00", "")
	// Block absolute paths.
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("absolute path %q not allowed", rel)
	}
	joined := filepath.Join(base, rel)
	cleaned := filepath.Clean(joined)
	// Verify the cleaned path is still under base.
	baseClean := filepath.Clean(base) + string(filepath.Separator)
	if !strings.HasPrefix(cleaned+string(filepath.Separator), baseClean) && cleaned != filepath.Clean(base) {
		return "", fmt.Errorf("path %q escapes base directory", rel)
	}
	return cleaned, nil
}
