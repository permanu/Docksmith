package docksmith

import (
	"path/filepath"
	"testing"
)

func TestCacheDir(t *testing.T) {
	base := "/var/cache/buildkit"
	cases := []struct {
		name  string
		appID string
		want  string
	}{
		{"valid id", "myapp-123", filepath.Join(base, "myapp-123")},
		{"path traversal", "../../etc", filepath.Join(base, "--etc")},
		{"empty id", "", filepath.Join(base, "unknown")},
		{"slashes", "foo/bar", filepath.Join(base, "foo-bar")},
		{"dots stripped", "foo.bar", filepath.Join(base, "foo.bar")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CacheDir(tc.appID)
			if got != tc.want {
				t.Errorf("CacheDir(%q) = %q, want %q", tc.appID, got, tc.want)
			}
		})
	}
}

func TestSanitizeAppID(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"valid alphanumeric-dash", "myapp-123", "myapp-123"},
		{"valid with underscore", "valid_ID", "valid_ID"},
		{"double-dot becomes unknown", "..", "unknown"},
		{"traversal path", "foo/../bar", "foo--bar"},
		{"slash replaced", "foo/bar", "foo-bar"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeAppID(tc.in)
			if got != tc.want {
				t.Errorf("sanitizeAppID(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestBuildkitCacheArgs(t *testing.T) {
	dir := CacheDir("myapp")
	args := BuildkitCacheArgs("myapp")
	if len(args) != 2 {
		t.Fatalf("want 2 args, got %d", len(args))
	}
	wantFrom := "--cache-from=type=local,src=" + dir
	wantTo := "--cache-to=type=local,dest=" + dir + ",mode=max"
	if args[0] != wantFrom {
		t.Errorf("args[0] = %q, want %q", args[0], wantFrom)
	}
	if args[1] != wantTo {
		t.Errorf("args[1] = %q, want %q", args[1], wantTo)
	}
}
