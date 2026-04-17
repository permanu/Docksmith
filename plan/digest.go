// digest.go — Wheel #1 Week 3 base-image digest resolution.
//
// ResolveBaseImageDigest queries a registry for the canonical content digest
// of an image reference. Permanu pins this digest in the BuildManifest so
// rebuilds are reproducible even if the upstream tag (e.g. node:22-alpine)
// is re-pushed.
//
// Network failures (offline, DNS, auth) are surfaced as wrapped errors with
// an empty digest return — the caller decides whether to fail the build or
// proceed with a best-effort manifest.
package plan

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/go-containerregistry/pkg/crane"
)

// ErrBaseImageUnresolvable is returned when the digest could not be fetched
// from the registry (network error, auth failure, unknown reference). Callers
// can errors.Is(err, ErrBaseImageUnresolvable) to decide whether to proceed
// with an empty digest or abort the build.
var ErrBaseImageUnresolvable = errors.New("base image digest unresolvable")

// ResolveBaseImageDigest returns the sha256:... digest for image by calling
// the equivalent of `crane digest <image>`. On success it returns a string
// prefixed with "sha256:". On failure it returns ("", wrappedErr) — the
// caller decides whether a missing digest blocks the build.
//
// This performs a network round-trip; callers in hot paths should cache the
// result. The underlying go-containerregistry client reads Docker credentials
// from the host environment (default keychain).
func ResolveBaseImageDigest(ctx context.Context, image string) (string, error) {
	if image == "" {
		return "", fmt.Errorf("%w: empty image reference", ErrBaseImageUnresolvable)
	}

	// crane.Digest currently takes no context; wrap in a goroutine so ctx
	// cancellation cancels the caller's wait (the underlying HTTP fetch
	// continues until its own deadline, but the caller is released).
	type result struct {
		digest string
		err    error
	}
	ch := make(chan result, 1)
	go func() {
		d, err := crane.Digest(image)
		ch <- result{digest: d, err: err}
	}()

	select {
	case <-ctx.Done():
		slog.Warn("ResolveBaseImageDigest: context canceled",
			slog.String("image", image),
			slog.String("err", ctx.Err().Error()),
		)
		return "", fmt.Errorf("%w: %w", ErrBaseImageUnresolvable, ctx.Err())
	case r := <-ch:
		if r.err != nil {
			return "", fmt.Errorf("%w: crane.Digest %q: %w", ErrBaseImageUnresolvable, image, r.err)
		}
		return r.digest, nil
	}
}
