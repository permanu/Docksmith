package docksmith

import (
	"github.com/permanu/docksmith/yamldef"
)

// Type aliases re-export yamldef types so existing callers keep working.
type FrameworkDef = yamldef.FrameworkDef
type DetectRules = yamldef.DetectRules
type DetectRule = yamldef.DetectRule
type VersionConfig = yamldef.VersionConfig
type VersionSource = yamldef.VersionSource
type PMConfig = yamldef.PMConfig
type PMSource = yamldef.PMSource
type PlanDef = yamldef.PlanDef
type StageDef = yamldef.StageDef
type StepDef = yamldef.StepDef
type CopyFromDef = yamldef.CopyFromDef
type DefaultsDef = yamldef.DefaultsDef
type TestCase = yamldef.TestCase
type TestExpect = yamldef.TestExpect
type TestResult = yamldef.TestResult
