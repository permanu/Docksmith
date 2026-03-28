package yamldef

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
	return FileExists(filepath.Join(dir, name))
}

// FileExists returns true when a non-directory entry exists at path.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func fileContains(path, substr string) bool {
	data, err := ReadFileLimited(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), substr)
}

// ReadFileLimited reads up to maxFileReadBytes from path.
// Returns an error if the file exceeds the limit.
func ReadFileLimited(path string) ([]byte, error) {
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

// ContainedPath joins base and rel, then verifies the result is under base.
// Prevents path traversal via "../" or absolute paths in rel.
func ContainedPath(base, rel string) (string, error) {
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

// IsYAMLFile returns true when the filename has a .yaml or .yml extension.
func IsYAMLFile(name string) bool {
	return strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")
}
