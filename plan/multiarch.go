// multiarch.go — Wheel #1 Week 4 multi-architecture build helpers.
//
// Docker buildx and buildkit can produce a manifest list (aka "multi-arch
// image") by accepting --platform=<csv>. The helpers here centralize the
// canonical platform list for Permanu builds (linux/amd64 + linux/arm64) and
// emit the flag sets used by the build pipeline.
//
// Cache hints are produced by BuildkitMultiArchCacheArgs — a per-platform
// sibling of BuildkitCacheArgs. buildkit's local cache exporter does not key
// on platform by itself, so concurrent amd64+arm64 builds sharing one cache
// dir would thrash. We fan out into per-arch sub-dirs instead.
package plan

import (
	"fmt"
	"strings"
)

// DefaultArchitectures returns the canonical Permanu target platforms for a
// multi-arch build: linux/amd64 and linux/arm64. The returned slice is a new
// allocation on each call — callers may mutate it safely.
func DefaultArchitectures() []string {
	return []string{"linux/amd64", "linux/arm64"}
}

// BuildxMultiArchArgs returns the buildx flags for producing a multi-arch
// image without pushing (--output=type=image,push=false). An empty platforms
// slice is normalized to DefaultArchitectures().
//
// Typical use — local or CI multi-arch validation that should not publish:
//
//	args := BuildxMultiArchArgs(nil)
//	// ["--platform=linux/amd64,linux/arm64", "--output=type=image,push=false"]
func BuildxMultiArchArgs(platforms []string) []string {
	platforms = normalizePlatforms(platforms)
	return []string{
		"--platform=" + strings.Join(platforms, ","),
		"--output=type=image,push=false",
	}
}

// BuildxPushArgs returns the buildx flags for building and pushing a
// multi-arch manifest list tagged as imageRef. An empty platforms slice is
// normalized to DefaultArchitectures(). imageRef is not validated — callers
// are responsible for passing a well-formed registry reference.
//
// Typical use — CI release flow:
//
//	args := BuildxPushArgs([]string{"linux/amd64"}, "ghcr.io/permanu/app:v1")
//	// ["--platform=linux/amd64", "--push", "--tag", "ghcr.io/permanu/app:v1"]
func BuildxPushArgs(platforms []string, imageRef string) []string {
	platforms = normalizePlatforms(platforms)
	return []string{
		"--platform=" + strings.Join(platforms, ","),
		"--push",
		"--tag",
		imageRef,
	}
}

// BuildkitMultiArchCacheArgs returns per-platform --cache-from / --cache-to
// flags so concurrent amd64+arm64 builds do not thrash a shared local cache
// directory. Each platform gets its own sub-dir under CacheDir(appID), keyed
// by a filesystem-safe slug (e.g. "linux-amd64").
//
// An empty platforms slice is normalized to DefaultArchitectures(). The
// returned slice preserves platform order: for each platform, --cache-from
// precedes --cache-to.
func BuildkitMultiArchCacheArgs(appID string, platforms []string) []string {
	platforms = normalizePlatforms(platforms)
	base := CacheDir(appID)
	args := make([]string, 0, len(platforms)*2)
	for _, p := range platforms {
		slug := platformSlug(p)
		dir := base + "/" + slug
		args = append(args,
			fmt.Sprintf("--cache-from=type=local,src=%s,platform=%s", dir, p),
			fmt.Sprintf("--cache-to=type=local,dest=%s,mode=max,platform=%s", dir, p),
		)
	}
	return args
}

// normalizePlatforms returns DefaultArchitectures() when platforms is empty;
// otherwise returns platforms unchanged. Never returns nil.
func normalizePlatforms(platforms []string) []string {
	if len(platforms) == 0 {
		return DefaultArchitectures()
	}
	return platforms
}

// platformSlug converts a Docker platform string ("linux/amd64") into a
// filesystem-safe slug ("linux-amd64"). Unknown characters are not filtered
// further because Docker platform strings are already constrained to
// [a-z0-9/._-].
func platformSlug(platform string) string {
	return strings.ReplaceAll(platform, "/", "-")
}
