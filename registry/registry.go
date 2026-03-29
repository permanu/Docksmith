package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

const maxHTTPResponseBytes = 10 << 20 // 10 MB

const DefaultRegistryURL = "https://raw.githubusercontent.com/permanu/docksmith-registry/main/index.json"

const registryCacheTTL = 24 * time.Hour

// allowInsecureHTTP is a test-only hook to permit http:// URLs in unit tests.
// Production code must never enable this.
var allowInsecureHTTP atomic.Bool

// SetAllowInsecureHTTP enables or disables the test-only insecure HTTP hook.
// This is intended exclusively for test code using httptest servers.
func SetAllowInsecureHTTP(v bool) { allowInsecureHTTP.Store(v) }

// isInsecureHTTPAllowed reports whether insecure HTTP URLs are currently permitted.
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
// Security: the index is fetched over TLS from the registryURL. If an attacker
// can MITM the TLS connection (or compromise the upstream repository), they can
// substitute both the download URL and the SHA256 in the index — the checksum
// only guards against download corruption, not index authenticity. Pinning the
// index itself would require a signature scheme (not yet implemented).
// TLS-only is enforced: non-HTTPS registry URLs are rejected in production.
func FetchIndex(registryURL string, offline bool) (*Index, error) {
	if !isInsecureHTTPAllowed() && !strings.HasPrefix(registryURL, "https://") {
		return nil, fmt.Errorf("registry: refusing non-HTTPS registry URL %q", registryURL)
	}

	cachePath, err := registryCachePath(registryURL)
	if err != nil {
		return nil, fmt.Errorf("registry cache path: %w", err)
	}

	if cached, ok := loadCachedIndex(cachePath); ok {
		return cached, nil
	}

	if offline {
		return nil, fmt.Errorf("registry: no cached index and --offline is set")
	}

	return fetchAndCache(registryURL, cachePath)
}

// Search finds entries whose name, runtime, or description contains query.
// Returns entries sorted by name.
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
// Verifies SHA256 checksum (required — entries without sha256 are rejected).
//
// Security: the SHA256 checksum guards against download corruption and partial
// tampering, but it comes from the same registry index as the URL. If the index
// itself is poisoned (MITM on TLS, compromised upstream repo), the attacker can
// change both fields. TLS-only enforcement on FetchIndex is the primary defence.
func InstallFramework(entry Entry) (string, error) {
	if entry.URL == "" {
		return "", fmt.Errorf("install %s: entry has no URL", entry.Name)
	}
	if !isInsecureHTTPAllowed() && !strings.HasPrefix(entry.URL, "https://") {
		return "", fmt.Errorf("install %s: refusing non-HTTPS download URL %q", entry.Name, entry.URL)
	}

	// Sanitize entry.Name to prevent path traversal.
	safeName := filepath.Base(entry.Name)
	if safeName != entry.Name || safeName == "." || safeName == ".." ||
		strings.ContainsAny(entry.Name, `/\`) || strings.Contains(entry.Name, "..") ||
		strings.Contains(entry.Name, "\x00") {
		return "", fmt.Errorf("install: invalid framework name %q", entry.Name)
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

	if entry.SHA256 == "" {
		return "", fmt.Errorf("install %s: registry entry missing sha256 checksum", safeName)
	}
	got := sha256.Sum256(data)
	gotHex := hex.EncodeToString(got[:])
	if gotHex != strings.ToLower(entry.SHA256) {
		return "", fmt.Errorf("install %s: sha256 mismatch: got %s, want %s", safeName, gotHex, entry.SHA256)
	}

	dest := filepath.Join(destDir, safeName+".yaml")

	// Final safety check: resolved path must be inside destDir.
	absDir, err2 := filepath.Abs(destDir)
	if err2 != nil {
		return "", fmt.Errorf("install %s: resolve dest dir: %w", safeName, err2)
	}
	absDest, err2 := filepath.Abs(dest)
	if err2 != nil {
		return "", fmt.Errorf("install %s: resolve dest path: %w", safeName, err2)
	}
	if !strings.HasPrefix(absDest, absDir+string(filepath.Separator)) {
		return "", fmt.Errorf("install %s: path escapes destination directory", safeName)
	}

	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return "", fmt.Errorf("install %s: write: %w", safeName, err)
	}
	return dest, nil
}

func loadCachedIndex(path string) (*Index, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	if time.Since(info.ModTime()) > registryCacheTTL {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, false
	}
	return &idx, true
}

func fetchAndCache(url, cachePath string) (*Index, error) {
	data, err := fetchURL(url)
	if err != nil {
		return nil, fmt.Errorf("registry: fetch index: %w", err)
	}
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("registry: parse index: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err == nil {
		_ = os.WriteFile(cachePath, data, 0o644)
	}
	return &idx, nil
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

func fetchURL(url string) ([]byte, error) {
	resp, err := httpClient.Get(url) //nolint:gosec
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	lr := io.LimitReader(resp.Body, maxHTTPResponseBytes+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxHTTPResponseBytes {
		return nil, fmt.Errorf("response from %s exceeds %d byte limit", url, maxHTTPResponseBytes)
	}
	return data, nil
}

// registryCachePath returns a cache file path keyed by registry URL to prevent
// cross-URL cache poisoning.
func registryCachePath(registryURL string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	h := sha256.Sum256([]byte(registryURL))
	name := "index-" + hex.EncodeToString(h[:]) + ".json"
	return filepath.Join(home, ".docksmith", "cache", name), nil
}

func userFrameworksDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".docksmith", "frameworks"), nil
}
