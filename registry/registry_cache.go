package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const registryCacheTTL = 24 * time.Hour

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

// loadCachedIndex reads and parses the cached index. Returns (index, fresh)
// where fresh indicates whether the cache is within TTL. A corrupt cache file
// is deleted and treated as a miss.
func loadCachedIndex(path string) (idx *Index, fresh bool) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var parsed Index
	if err := json.Unmarshal(data, &parsed); err != nil {
		// Corrupt cache — delete it.
		os.Remove(path)
		return nil, false
	}
	isFresh := time.Since(info.ModTime()) <= registryCacheTTL
	return &parsed, isFresh
}

// fetchAndCache downloads the index and writes it to the cache atomically.
// If the fetch fails and a stale cache exists, returns the stale data.
func fetchAndCache(url, cachePath string) (*Index, error) {
	data, err := fetchURL(url)
	if err != nil {
		// Offline fallback: if stale cache exists, use it.
		if stale, _ := loadCachedIndex(cachePath); stale != nil {
			return stale, nil
		}
		return nil, fmt.Errorf("registry: fetch index: %w", err)
	}
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("registry: parse index: %w", err)
	}
	writeAtomic(cachePath, data)
	return &idx, nil
}

// writeAtomic writes data to path using a temp file + rename. Errors are
// swallowed because cache writes are best-effort.
func writeAtomic(path string, data []byte) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
	}
}

// writeAtomicStrict is like writeAtomic but returns errors instead of swallowing them.
// Used for authoritative writes (e.g. framework install) where failure must be reported.
func writeAtomicStrict(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

func UserFrameworksDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".docksmith", "frameworks"), nil
}
