package docksmith

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// appIDSafe matches IDs that are already path-safe — no sanitization needed.
var appIDSafe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// CacheDir returns the BuildKit cache directory for an app.
// The appID is sanitized to prevent path traversal.
func CacheDir(appID string) string {
	return filepath.Join("/var/cache/buildkit", sanitizeAppID(appID))
}

// sanitizeAppID makes an app ID safe for use in file paths.
func sanitizeAppID(appID string) string {
	if appIDSafe.MatchString(appID) {
		return appID
	}
	safe := strings.ReplaceAll(appID, "\x00", "")
	// Loop until stable — single pass misses "....".
	for strings.Contains(safe, "..") {
		safe = strings.ReplaceAll(safe, "..", "")
	}
	safe = strings.ReplaceAll(safe, "/", "-")
	safe = strings.ReplaceAll(safe, "\\", "-")
	if safe == "" {
		safe = "unknown"
	}
	return safe
}

// BuildkitCacheArgs returns --cache-from and --cache-to flags for buildctl/docker buildx.
func BuildkitCacheArgs(appID string) []string {
	dir := CacheDir(appID)
	return []string{
		fmt.Sprintf("--cache-from=type=local,src=%s", dir),
		fmt.Sprintf("--cache-to=type=local,dest=%s,mode=max", dir),
	}
}
