package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
)

const DefaultRegistryURL = "https://raw.githubusercontent.com/permanu/docksmith-registry/main/index.json"

// allowInsecureHTTP is a test-only hook to permit http:// URLs in unit tests.
// Production code must never enable this.
var allowInsecureHTTP atomic.Bool

// SetAllowInsecureHTTP enables or disables the test-only insecure HTTP hook.
// This is intended exclusively for test code using httptest servers.
func SetAllowInsecureHTTP(v bool) { allowInsecureHTTP.Store(v) }

func isInsecureHTTPAllowed() bool { return allowInsecureHTTP.Load() }

// Index holds the framework registry metadata.
type Index struct {
	Version    int              `json:"version"`
	Frameworks map[string]Entry `json:"frameworks"`
}

// Entry describes a community-contributed framework definition.
type Entry struct {
	Version     string `json:"version"`
	Description string `json:"description"`
	Runtime     string `json:"runtime"`
	Author      string `json:"author"`
	SHA256      string `json:"sha256"`
	URL         string `json:"url"`

	// Name is injected after parsing the index map — not in the JSON.
	Name string `json:"-"`
}

// FetchIndex downloads and parses the registry index.
// Caches at ~/.docksmith/cache/<url-hash>.json with a 24h TTL, keyed by URL.
// Pass offline=true to use only the cache (no network call).
//
// Security: TLS-only is enforced. If fetch fails and a stale cache exists,
// returns the stale cache as offline fallback.
func FetchIndex(registryURL string, offline bool) (*Index, error) {
	if err := validateScheme(registryURL); err != nil {
		return nil, err
	}

	cachePath, err := registryCachePath(registryURL)
	if err != nil {
		return nil, fmt.Errorf("registry cache path: %w", err)
	}

	cached, fresh := loadCachedIndex(cachePath)
	if cached != nil && fresh {
		return cached, nil
	}

	if offline {
		if cached != nil {
			return cached, nil
		}
		return nil, fmt.Errorf("registry: no cached index and --offline is set")
	}

	return fetchAndCache(registryURL, cachePath)
}

// Search finds entries whose name, runtime, or description contains query.
// Case-insensitive. Returns entries sorted by name.
func Search(index *Index, query string) []Entry {
	q := strings.ToLower(query)
	var results []Entry
	for name, e := range index.Frameworks {
		e.Name = name
		if q == "" ||
			strings.Contains(strings.ToLower(name), q) ||
			strings.Contains(strings.ToLower(e.Runtime), q) ||
			strings.Contains(strings.ToLower(e.Description), q) {
			results = append(results, e)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})
	return results
}

// InstallFramework downloads a framework YAML to ~/.docksmith/frameworks/.
// Verifies SHA256 checksum (required). Uses atomic temp+rename for writes.
func InstallFramework(entry Entry) (string, error) {
	if entry.URL == "" {
		return "", fmt.Errorf("install %s: entry has no URL", entry.Name)
	}
	if err := validateScheme(entry.URL); err != nil {
		return "", fmt.Errorf("install %s: %w", entry.Name, err)
	}
	if entry.SHA256 == "" {
		return "", fmt.Errorf("install %s: registry entry missing sha256 checksum", entry.Name)
	}

	safeName, err := sanitizeFrameworkName(entry.Name)
	if err != nil {
		return "", err
	}

	destDir, err := userFrameworksDir()
	if err != nil {
		return "", fmt.Errorf("install %s: %w", safeName, err)
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("install %s: create dir: %w", safeName, err)
	}

	data, err := fetchURL(entry.URL)
	if err != nil {
		return "", fmt.Errorf("install %s: fetch: %w", safeName, err)
	}

	got := sha256.Sum256(data)
	gotHex := hex.EncodeToString(got[:])
	if gotHex != strings.ToLower(entry.SHA256) {
		return "", fmt.Errorf("install %s: sha256 mismatch: got %s, want %s", safeName, gotHex, entry.SHA256)
	}

	dest := filepath.Join(destDir, safeName+".yaml")
	if err := validateDestPath(dest, destDir); err != nil {
		return "", fmt.Errorf("install %s: %w", safeName, err)
	}

	writeAtomic(dest, data)

	// Verify the write actually landed.
	if _, err := os.Stat(dest); err != nil {
		return "", fmt.Errorf("install %s: write failed: %w", safeName, err)
	}
	return dest, nil
}

func sanitizeFrameworkName(name string) (string, error) {
	safe := filepath.Base(name)
	if safe != name || safe == "." || safe == ".." ||
		strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") ||
		strings.Contains(name, "\x00") {
		return "", fmt.Errorf("install: invalid framework name %q", name)
	}
	return safe, nil
}

func validateDestPath(dest, destDir string) error {
	absDir, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("resolve dest dir: %w", err)
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return fmt.Errorf("resolve dest path: %w", err)
	}
	if !strings.HasPrefix(absDest, absDir+string(filepath.Separator)) {
		return fmt.Errorf("path escapes destination directory")
	}
	return nil
}
