package core

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBuildManifestJSONRoundTrip(t *testing.T) {
	built := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		in   BuildManifest
	}{
		{
			name: "full_manifest",
			in: BuildManifest{
				SchemaVersion: "1.0",
				BuildID:       "018f2b1a-7a3f-7c2b-9e1a-4c1f2d3e4f50",
				Commit:        "a1b2c3d4e5f60718293a4b5c6d7e8f9012345678",
				CommitShort:   "a1b2c3d4",
				ReleaseName:   "amber-otter-42",
				BuiltAt:       built,
				Framework: FrameworkSnapshot{
					Name:     "nextjs",
					Version:  "14.2.1",
					Detector: "nextjs",
				},
				Runtime: RuntimeContract{
					Port:           3000,
					HealthPath:     "/healthz",
					HealthCmd:      "curl -fsS http://127.0.0.1:3000/healthz",
					ShutdownSignal: "SIGTERM",
					ShutdownGraceS: 10,
					RequiredEnv:    []string{"DATABASE_URL"},
					OptionalEnv:    []string{"SENTRY_DSN"},
				},
				BaseImage: BaseImageRef{
					Image:  "node:22-alpine",
					Digest: "sha256:deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
				},
				Dependencies: DependencyDigest{
					LockfileHashes: map[string]string{
						"package-lock.json": "sha256:cafebabecafebabecafebabecafebabecafebabecafebabecafebabecafebabe",
					},
					DirectCount: 42,
					TotalCount:  317,
				},
				SBOM:          json.RawMessage(`{"bomFormat":"CycloneDX","specVersion":"1.5"}`),
				ImageDigest:   "sha256:0011223344556677889900112233445566778899001122334455667788990011",
				Architectures: []string{"linux/amd64", "linux/arm64"},
			},
		},
		{
			name: "minimal_manifest_no_sbom_no_optional_env",
			in: BuildManifest{
				SchemaVersion: "1.0",
				BuildID:       "018f2b1a-7a3f-7c2b-9e1a-4c1f2d3e4f51",
				Commit:        "ffffffffffffffffffffffffffffffffffffffff",
				CommitShort:   "ffffffff",
				ReleaseName:   "blue-bison-1",
				BuiltAt:       built,
				Framework: FrameworkSnapshot{
					Name:     "go",
					Version:  "1.26",
					Detector: "go",
				},
				Runtime: RuntimeContract{
					Port:           8080,
					HealthPath:     "/healthz",
					ShutdownSignal: "SIGTERM",
					ShutdownGraceS: 10,
					RequiredEnv:    []string{},
				},
				BaseImage: BaseImageRef{
					Image:  "gcr.io/distroless/static:nonroot",
					Digest: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
				},
				Dependencies: DependencyDigest{
					LockfileHashes: map[string]string{
						"go.sum": "sha256:2222222222222222222222222222222222222222222222222222222222222222",
					},
					DirectCount: 5,
					TotalCount:  18,
				},
				ImageDigest:   "sha256:3333333333333333333333333333333333333333333333333333333333333333",
				Architectures: []string{"linux/amd64"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.in)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}

			var got BuildManifest
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}

			// Re-marshal to compare deterministically — time.Time and
			// json.RawMessage do not compare cleanly with reflect.DeepEqual
			// across round trips in every Go version.
			reData, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("re-Marshal: %v", err)
			}
			if string(data) != string(reData) {
				t.Errorf("round-trip mismatch:\n first=%s\nsecond=%s", data, reData)
			}
		})
	}
}

func TestBuildManifestJSONTagsAndRequiredFields(t *testing.T) {
	m := BuildManifest{
		SchemaVersion: "1.0",
		BuildID:       "bid",
		Commit:        "c",
		CommitShort:   "cs",
		ReleaseName:   "rn",
		BuiltAt:       time.Unix(0, 0).UTC(),
		Framework:     FrameworkSnapshot{Name: "go", Version: "1.26", Detector: "go"},
		Runtime: RuntimeContract{
			Port:           8080,
			HealthPath:     "/healthz",
			ShutdownSignal: "SIGTERM",
			ShutdownGraceS: 10,
			RequiredEnv:    []string{"X"},
		},
		BaseImage:     BaseImageRef{Image: "i", Digest: "d"},
		Dependencies:  DependencyDigest{LockfileHashes: map[string]string{"go.sum": "sha256:x"}, DirectCount: 1, TotalCount: 1},
		ImageDigest:   "sha256:img",
		Architectures: []string{"linux/amd64"},
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(data)

	// The substrate contract freezes these JSON keys. If any of these asserts
	// fires, the Permanu and Dwaar tracks break — update the contract doc first.
	mustContain := []string{
		`"schema_version":"1.0"`,
		`"build_id":"bid"`,
		`"commit":"c"`,
		`"commit_short":"cs"`,
		`"release_name":"rn"`,
		`"built_at":`,
		`"framework":`,
		`"runtime":`,
		`"base_image":`,
		`"dependencies":`,
		`"image_digest":"sha256:img"`,
		`"architectures":["linux/amd64"]`,
		`"port":8080`,
		`"health_path":"/healthz"`,
		`"shutdown_signal":"SIGTERM"`,
		`"shutdown_grace_s":10`,
		`"required_env":["X"]`,
		`"lockfile_hashes":{"go.sum":"sha256:x"}`,
		`"direct_count":1`,
		`"total_count":1`,
	}
	for _, want := range mustContain {
		if !strings.Contains(s, want) {
			t.Errorf("missing tag %q in output:\n%s", want, s)
		}
	}

	// Omitempty fields must NOT appear when empty.
	mustNotContain := []string{
		`"sbom":`,
		`"health_cmd":`,
		`"optional_env":`,
	}
	for _, bad := range mustNotContain {
		if strings.Contains(s, bad) {
			t.Errorf("unexpected field %q in output:\n%s", bad, s)
		}
	}
}

func TestManifestSHA(t *testing.T) {
	built := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	base := BuildManifest{
		SchemaVersion: "1.0",
		BuildID:       "018f2b1a-7a3f-7c2b-9e1a-4c1f2d3e4f50",
		Commit:        "a1b2c3d4e5f60718293a4b5c6d7e8f9012345678",
		CommitShort:   "a1b2c3d4",
		ReleaseName:   "amber-otter-42",
		BuiltAt:       built,
		Framework:     FrameworkSnapshot{Name: "nextjs", Version: "14.2.1", Detector: "nextjs"},
		Runtime: RuntimeContract{
			Port:           3000,
			HealthPath:     "/healthz",
			ShutdownSignal: "SIGTERM",
			ShutdownGraceS: 10,
			RequiredEnv:    []string{"DATABASE_URL"},
		},
		BaseImage:     BaseImageRef{Image: "node:22-alpine", Digest: "sha256:deadbeef"},
		Dependencies:  DependencyDigest{LockfileHashes: map[string]string{"package-lock.json": "sha256:cafebabe"}, DirectCount: 1, TotalCount: 1},
		ImageDigest:   "sha256:0011",
		Architectures: []string{"linux/amd64"},
	}

	t.Run("prefix_and_length", func(t *testing.T) {
		got, err := ManifestSHA(base)
		if err != nil {
			t.Fatalf("ManifestSHA: %v", err)
		}
		if !strings.HasPrefix(got, "sha256:") {
			t.Errorf("want sha256: prefix, got %q", got)
		}
		// sha256 hex is 64 chars; plus "sha256:" prefix (7) = 71.
		if len(got) != 71 {
			t.Errorf("want length 71, got %d (%q)", len(got), got)
		}
	})

	t.Run("deterministic", func(t *testing.T) {
		a, err := ManifestSHA(base)
		if err != nil {
			t.Fatalf("ManifestSHA a: %v", err)
		}
		b, err := ManifestSHA(base)
		if err != nil {
			t.Fatalf("ManifestSHA b: %v", err)
		}
		if a != b {
			t.Errorf("ManifestSHA not deterministic: %q vs %q", a, b)
		}
	})

	t.Run("sensitive_to_changes", func(t *testing.T) {
		base1, err := ManifestSHA(base)
		if err != nil {
			t.Fatalf("ManifestSHA base1: %v", err)
		}
		mutated := base
		mutated.ImageDigest = "sha256:different"
		base2, err := ManifestSHA(mutated)
		if err != nil {
			t.Fatalf("ManifestSHA mutated: %v", err)
		}
		if base1 == base2 {
			t.Errorf("ManifestSHA unchanged after field mutation: %q", base1)
		}
	})

	t.Run("map_key_order_stable", func(t *testing.T) {
		// Rebuild the manifest with a map populated in reverse insertion order.
		// encoding/json sorts map keys, so the SHA must not depend on build order.
		m1 := base
		m1.Dependencies = DependencyDigest{
			LockfileHashes: map[string]string{
				"a.lock": "sha256:1",
				"b.lock": "sha256:2",
				"c.lock": "sha256:3",
			},
			DirectCount: 1,
			TotalCount:  3,
		}
		m2 := base
		m2.Dependencies = DependencyDigest{
			LockfileHashes: map[string]string{
				"c.lock": "sha256:3",
				"b.lock": "sha256:2",
				"a.lock": "sha256:1",
			},
			DirectCount: 1,
			TotalCount:  3,
		}
		h1, err := ManifestSHA(m1)
		if err != nil {
			t.Fatalf("ManifestSHA m1: %v", err)
		}
		h2, err := ManifestSHA(m2)
		if err != nil {
			t.Fatalf("ManifestSHA m2: %v", err)
		}
		if h1 != h2 {
			t.Errorf("ManifestSHA sensitive to map insertion order: %q vs %q", h1, h2)
		}
	})
}
