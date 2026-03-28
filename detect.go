package docksmith

// NamedDetector pairs a name with a detection function for registry ordering.
type NamedDetector struct {
	Name   string
	Detect DetectorFunc
}

// DetectOptions configures detection behavior.
type DetectOptions struct {
	// ConfigFileNames lists filenames to treat as user config (e.g. "docksmith.toml").
	// Config loading is implemented in #DS-016.
	ConfigFileNames []string
}

// detectors is the ordered registry. Individual runtime detectors (#DS-006–#DS-014)
// append to this at init time. Dockerfile and static fallback are handled inline.
var detectors []NamedDetector

// Detect analyzes dir and returns the detected framework.
// Returns a static-site framework as fallback if nothing matches.
func Detect(dir string) (*Framework, error) {
	return DetectWithOptions(dir, DetectOptions{})
}

// DetectWithOptions runs detection with custom options.
// Returns an error when a config file exists but is invalid — this prevents
// silent fallthrough to auto-detection when the user intended a specific config.
func DetectWithOptions(dir string, opts DetectOptions) (*Framework, error) {
	// User config — highest priority of all.
	fw, cfgErr := loadConfigFramework(dir, opts)
	if cfgErr != nil {
		return nil, cfgErr
	}
	if fw != nil {
		return fw, nil
	}

	// User Dockerfile — second priority before auto-detection.
	if hasFile(dir, "Dockerfile") {
		return &Framework{Name: "dockerfile", Port: 8080}, nil
	}

	for _, nd := range detectors {
		if fw := nd.Detect(dir); fw != nil {
			return fw, nil
		}
	}

	return &Framework{Name: "static", Port: 80, OutputDir: "."}, nil
}

// loadConfigFramework loads the user config and converts it to a Framework.
// Returns (nil, nil) when no config file exists.
// Returns (nil, err) when a config file exists but is invalid.
func loadConfigFramework(dir string, opts DetectOptions) (*Framework, error) {
	names := opts.ConfigFileNames
	if len(names) == 0 {
		names = defaultConfigFileNames
	}
	cfg, err := loadConfigWithNames(dir, names)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil
	}
	return cfg.ToFramework(), nil
}

// RegisterDetector prepends d to the registry, giving it the highest priority
// among registered detectors.
func RegisterDetector(name string, d DetectorFunc) {
	detectors = append([]NamedDetector{{name, d}}, detectors...)
}

// RegisterDetectorBefore inserts d immediately before the named detector.
// If before is not found, d is prepended.
func RegisterDetectorBefore(before, name string, d DetectorFunc) {
	for i, nd := range detectors {
		if nd.Name == before {
			updated := make([]NamedDetector, 0, len(detectors)+1)
			updated = append(updated, detectors[:i]...)
			updated = append(updated, NamedDetector{name, d})
			updated = append(updated, detectors[i:]...)
			detectors = updated
			return
		}
	}
	RegisterDetector(name, d)
}
