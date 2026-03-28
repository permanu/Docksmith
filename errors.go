package docksmith

import "github.com/permanu/docksmith/core"

// Error aliases re-export core sentinel errors for backward compatibility.
var (
	ErrNotDetected  = core.ErrNotDetected
	ErrInvalidConfig = core.ErrInvalidConfig
	ErrInvalidPlan  = core.ErrInvalidPlan
)
