package docksmith

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxHTTPResponseBytes = 10 << 20 // 10 MB

const DefaultRegistryURL = "https://raw.githubusercontent.com/permanu/docksmith-registry/main/index.json"

const registryCacheTTL = 24 * time.Hour

// RegistryIndex holds the framework registry metadata.
type RegistryIndex struct {
	Version    int                      `json:"version"`
	Frameworks map[string]RegistryEntry `json:"frameworks"`
}

// RegistryEntry describes a community-contributed framework definition.
type RegistryEntry struct {
	Version     string `json:"version"`
	Description string `json:"description"`
	Runtime     string `json:"runtime"`
	Author      string `json:"author"`
	SHA256      string `json:"sha256"`
	URL         string `json:"url"`

	// Name is injected after parsing the index map — not in the JSON.
	Name string `json:"-"`
}

// FetchRegistryIndex downloads and parses the registry index.
// Caches at ~/.docksmith/cache/index.json with a 24h TTL.
// Pass offline=true to use only the cache (no network call).
func FetchRegistryIndex(registryURL string, offline bool) (*RegistryIndex, error) {
	cachePath, err := registryCachePath()
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

// SearchRegistry finds entries whose name, runtime, or description contains query.
// Returns entries sorted by name (map iteration order is random).
func SearchRegistry(index *RegistryIndex, query string) []RegistryEntry {
	q := strings.ToLower(query)
	var results []RegistryEntry
	for name, e := range index.Frameworks {
		e.Name = name
		if q == "" ||
			strings.Contains(strings.ToLower(name), q) ||
			strings.Contains(strings.ToLower(e.Runtime), q) ||
			strings.Contains(strings.ToLower(e.Description), q) {
			results = append(results, e)
		}
	}
	return results
}

// InstallFramework downloads a framework YAML to ~/.docksmith/frameworks/.
// Verifies SHA256 checksum when provided in the registry entry.
func InstallFramework(entry RegistryEntry) (string, error) {
	if entry.URL == "" {
		return "", fmt.Errorf("install %s: entry has no URL", entry.Name)
	}

	destDir, err := userFrameworksDir()
	if err != nil {
		return "", fmt.Errorf("install %s: %w", entry.Name, err)
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("install %s: create dir: %w", entry.Name, err)
	}

	data, err := fetchURL(entry.URL)
	if err != nil {
		return "", fmt.Errorf("install %s: fetch: %w", entry.Name, err)
	}

	if entry.SHA256 == "" {
		return "", fmt.Errorf("install %s: registry entry missing sha256 checksum", entry.Name)
	}
	got := sha256.Sum256(data)
	gotHex := hex.EncodeToString(got[:])
	if gotHex != strings.ToLower(entry.SHA256) {
		return "", fmt.Errorf("install %s: sha256 mismatch: got %s, want %s", entry.Name, gotHex, entry.SHA256)
	}

	dest := filepath.Join(destDir, entry.Name+".yaml")
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return "", fmt.Errorf("install %s: write: %w", entry.Name, err)
	}
	return dest, nil
}

func loadCachedIndex(path string) (*RegistryIndex, bool) {
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
	var idx RegistryIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, false
	}
	return &idx, true
}

func fetchAndCache(url, cachePath string) (*RegistryIndex, error) {
	data, err := fetchURL(url)
	if err != nil {
		return nil, fmt.Errorf("registry: fetch index: %w", err)
	}
	var idx RegistryIndex
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

func registryCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".docksmith", "cache", "index.json"), nil
}

func userFrameworksDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".docksmith", "frameworks"), nil
}
