package docksmith

import "github.com/permanu/docksmith/core"

// Type aliases re-export core plan types for backward compatibility.
type StepType = core.StepType

const (
	StepWorkdir    = core.StepWorkdir
	StepCopy       = core.StepCopy
	StepCopyFrom   = core.StepCopyFrom
	StepRun        = core.StepRun
	StepEnv        = core.StepEnv
	StepArg        = core.StepArg
	StepExpose     = core.StepExpose
	StepCmd        = core.StepCmd
	StepEntrypoint = core.StepEntrypoint
	StepUser       = core.StepUser
	StepHealthcheck = core.StepHealthcheck
)

type BuildPlan = core.BuildPlan
type Stage = core.Stage
type Step = core.Step
type CacheMount = core.CacheMount
type SecretMount = core.SecretMount
type CopyFrom = core.CopyFrom
