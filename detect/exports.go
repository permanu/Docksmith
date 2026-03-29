package detect

// Exported wrappers for helper functions used by tests in other packages.

// HasFile reports whether a regular file with the given name exists in dir.
func HasFile(dir, name string) bool { return hasFile(dir, name) }

// FileExists reports whether path exists and is a regular file.
func FileExists(path string) bool { return fileExists(path) }

// FileContains reports whether the file at path contains substr.
func FileContains(path, substr string) bool { return fileContains(path, substr) }

// ParseVersionString extracts a version number from a raw version string.
func ParseVersionString(s string) string { return parseVersionString(s) }

// ExtractMajorVersion pulls a usable version from semver constraints.
func ExtractMajorVersion(constraint string) string { return extractMajorVersion(constraint) }

// ContainedPath joins base and rel, verifying the result stays under base.
func ContainedPath(base, rel string) (string, error) { return containedPath(base, rel) }

// GetDetectors returns the current detector registry. For testing only.
func GetDetectors() []NamedDetector {
	detectorsMu.RLock()
	defer detectorsMu.RUnlock()
	out := make([]NamedDetector, len(detectors))
	copy(out, detectors)
	return out
}

// SetDetectors replaces the detector registry. For testing only.
func SetDetectors(d []NamedDetector) {
	detectorsMu.Lock()
	detectors = d
	detectorsMu.Unlock()
}
