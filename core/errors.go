package core

import "errors"

var (
	ErrNotDetected   = errors.New("no framework detected")
	ErrInvalidConfig = errors.New("invalid config")
	ErrInvalidPlan   = errors.New("invalid build plan")
)
