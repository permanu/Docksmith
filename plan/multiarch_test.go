package plan

import (
	"reflect"
	"strings"
	"testing"
)

func TestDefaultArchitectures(t *testing.T) {
	got := DefaultArchitectures()
	want := []string{"linux/amd64", "linux/arm64"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DefaultArchitectures() = %v, want %v", got, want)
	}

	// Must return an independent slice — mutating the result must not affect
	// a subsequent call.
	got[0] = "linux/evil"
	fresh := DefaultArchitectures()
	if fresh[0] != "linux/amd64" {
		t.Errorf("DefaultArchitectures shares backing array: fresh[0] = %q", fresh[0])
	}
}

func TestBuildxMultiArchArgs(t *testing.T) {
	cases := []struct {
		name      string
		platforms []string
		want      []string
	}{
		{
			name:      "nil falls back to defaults",
			platforms: nil,
			want: []string{
				"--platform=linux/amd64,linux/arm64",
				"--output=type=image,push=false",
			},
		},
		{
			name:      "empty falls back to defaults",
			platforms: []string{},
			want: []string{
				"--platform=linux/amd64,linux/arm64",
				"--output=type=image,push=false",
			},
		},
		{
			name:      "single platform",
			platforms: []string{"linux/arm64"},
			want: []string{
				"--platform=linux/arm64",
				"--output=type=image,push=false",
			},
		},
		{
			name:      "custom multi-platform order preserved",
			platforms: []string{"linux/arm/v7", "linux/amd64"},
			want: []string{
				"--platform=linux/arm/v7,linux/amd64",
				"--output=type=image,push=false",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildxMultiArchArgs(tc.platforms)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("BuildxMultiArchArgs(%v) = %v, want %v", tc.platforms, got, tc.want)
			}
		})
	}
}

func TestBuildxPushArgs(t *testing.T) {
	cases := []struct {
		name      string
		platforms []string
		imageRef  string
		want      []string
	}{
		{
			name:      "defaults when empty",
			platforms: nil,
			imageRef:  "ghcr.io/permanu/app:v1",
			want: []string{
				"--platform=linux/amd64,linux/arm64",
				"--push",
				"--tag",
				"ghcr.io/permanu/app:v1",
			},
		},
		{
			name:      "single platform preserved",
			platforms: []string{"linux/amd64"},
			imageRef:  "registry.example.com/app:sha-abc",
			want: []string{
				"--platform=linux/amd64",
				"--push",
				"--tag",
				"registry.example.com/app:sha-abc",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildxPushArgs(tc.platforms, tc.imageRef)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("BuildxPushArgs(%v, %q) = %v, want %v", tc.platforms, tc.imageRef, got, tc.want)
			}
		})
	}
}

func TestBuildkitMultiArchCacheArgs(t *testing.T) {
	appID := "myapp"
	base := CacheDir(appID)

	t.Run("defaults when empty", func(t *testing.T) {
		got := BuildkitMultiArchCacheArgs(appID, nil)
		if len(got) != 4 {
			t.Fatalf("want 4 args (2 platforms x 2 flags), got %d: %v", len(got), got)
		}
		// amd64 first, arm64 second.
		checkPair(t, got[0], got[1], base+"/linux-amd64", "linux/amd64")
		checkPair(t, got[2], got[3], base+"/linux-arm64", "linux/arm64")
	})

	t.Run("single platform", func(t *testing.T) {
		got := BuildkitMultiArchCacheArgs(appID, []string{"linux/arm64"})
		if len(got) != 2 {
			t.Fatalf("want 2 args, got %d: %v", len(got), got)
		}
		checkPair(t, got[0], got[1], base+"/linux-arm64", "linux/arm64")
	})

	t.Run("sanitizes app id", func(t *testing.T) {
		got := BuildkitMultiArchCacheArgs("../../etc/passwd", []string{"linux/amd64"})
		if len(got) != 2 {
			t.Fatalf("want 2 args, got %d", len(got))
		}
		// Path must not contain traversal — SanitizeAppID strips "..".
		if strings.Contains(got[0], "..") || strings.Contains(got[1], "..") {
			t.Errorf("path traversal not sanitized: %v", got)
		}
	})

	t.Run("preserves caller order", func(t *testing.T) {
		got := BuildkitMultiArchCacheArgs(appID, []string{"linux/arm64", "linux/amd64"})
		if len(got) != 4 {
			t.Fatalf("want 4 args, got %d", len(got))
		}
		checkPair(t, got[0], got[1], base+"/linux-arm64", "linux/arm64")
		checkPair(t, got[2], got[3], base+"/linux-amd64", "linux/amd64")
	})
}

func checkPair(t *testing.T, from, to, dir, platform string) {
	t.Helper()
	wantFrom := "--cache-from=type=local,src=" + dir + ",platform=" + platform
	wantTo := "--cache-to=type=local,dest=" + dir + ",mode=max,platform=" + platform
	if from != wantFrom {
		t.Errorf("cache-from = %q, want %q", from, wantFrom)
	}
	if to != wantTo {
		t.Errorf("cache-to = %q, want %q", to, wantTo)
	}
}
