package docksmith

import (
	"cmp"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/permanu/docksmith/core"
	"github.com/permanu/docksmith/detect"
	"github.com/permanu/docksmith/registry"
	"github.com/permanu/docksmith/yamldef"
)

// AutoFetchOptions configures the registry auto-fetch callback returned by
// NewAutoFetch. Pass the result as DetectOptions.AutoFetch.
type AutoFetchOptions struct {
	RegistryURL    string
	Interactive    bool
	ConfirmInstall func(name, description string) bool
}

// NewAutoFetch returns a callback suitable for DetectOptions.AutoFetch.
// It searches the community registry, installs a matching framework, reloads
// YAML defs, and re-detects. Returns (nil, nil) on miss or network failure.
func NewAutoFetch(afo AutoFetchOptions) func(dir string) (*core.Framework, error) {
	return func(dir string) (*core.Framework, error) {
		query := detect.SearchQueryFromDir(dir)
		if query == "" {
			return nil, nil
		}

		url := cmp.Or(afo.RegistryURL, registry.DefaultRegistryURL)

		idx, err := registry.FetchIndex(url, false)
		if err != nil {
			slog.Debug("registry fetch failed, falling back", "err", err)
			return nil, nil
		}

		results := registry.Search(idx, query)
		if len(results) == 0 {
			return nil, nil
		}

		entry := results[0]

		if alreadyInstalled(entry.Name) {
			return nil, nil
		}

		if afo.Interactive && afo.ConfirmInstall != nil {
			if !afo.ConfirmInstall(entry.Name, entry.Description) {
				return nil, nil
			}
		}

		destPath, err := registry.InstallFramework(entry)
		if err != nil {
			slog.Debug("registry install failed, falling back", "err", err)
			return nil, nil
		}

		fw, err := loadInstalledAndDetect(destPath, dir)
		if err != nil {
			return nil, fmt.Errorf("auto-fetch: %w", err)
		}
		return fw, nil
	}
}

func alreadyInstalled(name string) bool {
	fwDir, err := registry.UserFrameworksDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(fwDir, name+".yaml"))
	return err == nil
}

func loadInstalledAndDetect(yamlPath, dir string) (*core.Framework, error) {
	defs, err := yamldef.LoadFrameworkDefs(filepath.Dir(yamlPath))
	if err != nil {
		return nil, err
	}
	for _, def := range defs {
		name, port, matched := yamldef.EvalDefAgainstDir(def, dir)
		if matched {
			return &core.Framework{Name: name, Port: port}, nil
		}
	}
	return nil, nil
}
