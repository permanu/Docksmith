package yamldef

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadFrameworkDefs loads YAML framework definitions from dir.
// Files that fail to parse are skipped with an error returned after all files
// are attempted. The caller decides whether partial results are acceptable.
func LoadFrameworkDefs(dir string) ([]*FrameworkDef, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read framework dir %s: %w", dir, err)
	}

	var defs []*FrameworkDef
	var errs []string
	for _, e := range entries {
		if e.IsDir() || !IsYAMLFile(e.Name()) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", e.Name(), err))
			continue
		}
		var def FrameworkDef
		if err := yaml.Unmarshal(data, &def); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", e.Name(), err))
			continue
		}
		if def.Name == "" {
			errs = append(errs, fmt.Sprintf("%s: missing name field", e.Name()))
			continue
		}
		defs = append(defs, &def)
	}

	if len(errs) > 0 {
		return defs, fmt.Errorf("framework load errors:\n  %s", strings.Join(errs, "\n  "))
	}
	return defs, nil
}
