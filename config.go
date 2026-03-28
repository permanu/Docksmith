package docksmith

import (
	"github.com/permanu/docksmith/config"
)

// Type aliases re-export config types so existing callers keep working.
type Config = config.Config
type BuildConfig = config.BuildConfig
type StartConfig = config.StartConfig
type InstallConfig = config.InstallConfig
type RuntimeCfg = config.RuntimeCfg

// LoadConfig reads the first matching config file from dir.
// Returns (nil, nil) if no config file exists.
func LoadConfig(dir string) (*Config, error) {
	return config.Load(dir)
}

// LoadPlanOptions reads the config from dir and converts it to a PlanOption slice.
// Returns nil (not an error) when no config file exists.
func LoadPlanOptions(dir string) ([]PlanOption, error) {
	cfg, err := config.Load(dir)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil
	}
	return ConfigToPlanOptions(cfg)
}
