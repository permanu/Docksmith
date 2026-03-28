package docksmith

import "github.com/permanu/docksmith/core"

// Type aliases re-export core types for backward compatibility.
type Framework = core.Framework
type DetectorFunc = core.DetectorFunc

// Function aliases re-export core functions for backward compatibility.
var FrameworkFromJSON = core.FrameworkFromJSON
