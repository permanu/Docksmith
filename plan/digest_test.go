package plan

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestResolveBaseImageDigestEmptyRef(t *testing.T) {
	_, err := ResolveBaseImageDigest(context.Background(), "")
	if err == nil {
		t.Fatalf("expected error for empty image reference, got nil")
	}
	if !errors.Is(err, ErrBaseImageUnresolvable) {
		t.Errorf("expected ErrBaseImageUnresolvable, got %v", err)
	}
	if !strings.Contains(err.Error(), "empty image reference") {
		t.Errorf("expected 'empty image reference' in err, got %q", err.Error())
	}
}

func TestResolveBaseImageDigestContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before calling so we hit the cancellation path
	_, err := ResolveBaseImageDigest(ctx, "gcr.io/distroless/static:nonroot")
	if err == nil {
		t.Fatalf("expected error for canceled context, got nil")
	}
	if !errors.Is(err, ErrBaseImageUnresolvable) {
		t.Errorf("expected wrapped ErrBaseImageUnresolvable, got %v", err)
	}
}

// TestResolveBaseImageDigestLive hits a real registry. Skipped with -short
// and on any network failure — this is a smoke test, not a correctness gate.
func TestResolveBaseImageDigestLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live registry test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	digest, err := ResolveBaseImageDigest(ctx, "gcr.io/distroless/static:nonroot")
	if err != nil {
		t.Skipf("registry unreachable (expected in offline CI): %v", err)
	}
	if !strings.HasPrefix(digest, "sha256:") {
		t.Errorf("expected sha256: prefix, got %q", digest)
	}
	// sha256 hex is 64 chars; plus "sha256:" prefix.
	if len(digest) != len("sha256:")+64 {
		t.Errorf("unexpected digest length %d: %q", len(digest), digest)
	}
}
