package docksmith

import "github.com/permanu/docksmith/registry"

// Backward-compatible aliases for the registry package types and functions.
// New code should import github.com/permanu/docksmith/registry directly.

const DefaultRegistryURL = registry.DefaultRegistryURL

type RegistryIndex = registry.Index
type RegistryEntry = registry.Entry

var FetchRegistryIndex = registry.FetchIndex
var SearchRegistry = registry.Search
var InstallFramework = registry.InstallFramework
