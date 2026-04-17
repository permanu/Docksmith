package emit

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/permanu/docksmith/core"
)

func fullManifest() core.BuildManifest {
	return core.BuildManifest{
		SchemaVersion: "1.0",
		BuildID:       "018f2b1a-7a3f-7c2b-9e1a-4c1f2d3e4f50",
		Commit:        "a1b2c3d4e5f60718293a4b5c6d7e8f9012345678",
		CommitShort:   "a1b2c3d4",
		ReleaseName:   "amber-otter-42",
		BuiltAt:       time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC),
		Framework: core.FrameworkSnapshot{
			Name:     "nextjs",
			Version:  "14.2.1",
			Detector: "nextjs",
		},
		Runtime: core.RuntimeContract{
			Port:           3000,
			HealthPath:     "/healthz",
			HealthCmd:      "curl -fsS http://127.0.0.1:3000/healthz",
			ShutdownSignal: "SIGTERM",
			ShutdownGraceS: 10,
			RequiredEnv:    []string{"DATABASE_URL"},
			OptionalEnv:    []string{"SENTRY_DSN"},
		},
		BaseImage: core.BaseImageRef{
			Image:  "node:22-alpine",
			Digest: "sha256:deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
		},
		Dependencies: core.DependencyDigest{
			LockfileHashes: map[string]string{
				"package-lock.json": "sha256:cafebabecafebabecafebabecafebabecafebabecafebabecafebabecafebabe",
			},
			DirectCount: 42,
			TotalCount:  317,
		},
		SBOM:          json.RawMessage(`{"bomFormat":"CycloneDX","specVersion":"1.5"}`),
		ImageDigest:   "sha256:0011223344556677889900112233445566778899001122334455667788990011",
		Architectures: []string{"linux/amd64", "linux/arm64"},
	}
}

func TestBuildLabelsContainsAllContractFields(t *testing.T) {
	m := fullManifest()
	lines := BuildLabels(m)
	joined := strings.Join(lines, "")

	// Every field in the substrate contract's OCI label table must produce
	// a LABEL line.
	mustContain := []string{
		`LABEL io.permanu.manifest.schema="1.0"`,
		`LABEL io.permanu.manifest.id="018f2b1a-7a3f-7c2b-9e1a-4c1f2d3e4f50"`,
		`LABEL io.permanu.manifest.sha="sha256:`,
		`LABEL io.permanu.framework.name="nextjs"`,
		`LABEL io.permanu.framework.version="14.2.1"`,
		`LABEL io.permanu.build.commit="a1b2c3d4e5f60718293a4b5c6d7e8f9012345678"`,
		`LABEL io.permanu.build.release_name="amber-otter-42"`,
		`LABEL io.permanu.runtime.port="3000"`,
		`LABEL io.permanu.runtime.health_path="/healthz"`,
		`LABEL io.permanu.runtime.shutdown_signal="SIGTERM"`,
		`LABEL io.permanu.base.image="node:22-alpine"`,
		`LABEL io.permanu.base.digest="sha256:deadbeef`,
		`LABEL io.permanu.manifest.json=`,
	}
	for _, want := range mustContain {
		if !strings.Contains(joined, want) {
			t.Errorf("missing label line %q in output:\n%s", want, joined)
		}
	}
}

func TestBuildLabelsOmitsOptionalEmptyFields(t *testing.T) {
	// A bare manifest with no build-id, no framework, no commit, no base image.
	// Only io.permanu.manifest.schema and io.permanu.manifest.json must appear.
	m := core.BuildManifest{SchemaVersion: "1.0"}
	lines := BuildLabels(m)
	joined := strings.Join(lines, "")

	mustContain := []string{
		`LABEL io.permanu.manifest.schema="1.0"`,
		`LABEL io.permanu.manifest.json=`,
	}
	for _, want := range mustContain {
		if !strings.Contains(joined, want) {
			t.Errorf("missing required label %q:\n%s", want, joined)
		}
	}

	// No optional labels should leak out when the manifest is empty.
	mustNotContain := []string{
		`io.permanu.manifest.id=`,
		`io.permanu.framework.name=`,
		`io.permanu.framework.version=`,
		`io.permanu.build.commit=`,
		`io.permanu.build.release_name=`,
		`io.permanu.runtime.port=`,
		`io.permanu.runtime.health_path=`,
		`io.permanu.runtime.shutdown_signal=`,
		`io.permanu.base.image=`,
		`io.permanu.base.digest=`,
	}
	for _, bad := range mustNotContain {
		if strings.Contains(joined, bad) {
			t.Errorf("unexpected label %q emitted for empty manifest:\n%s", bad, joined)
		}
	}
}

func TestBuildLabelsJSONRoundTrips(t *testing.T) {
	m := fullManifest()
	lines := BuildLabels(m)

	// Find the io.permanu.manifest.json line and extract the quoted value.
	var jsonLine string
	for _, l := range lines {
		if strings.HasPrefix(l, "LABEL io.permanu.manifest.json=") {
			jsonLine = l
			break
		}
	}
	if jsonLine == "" {
		t.Fatalf("no io.permanu.manifest.json label emitted")
	}

	const prefix = "LABEL io.permanu.manifest.json="
	quoted := strings.TrimSuffix(strings.TrimPrefix(jsonLine, prefix), "\n")
	raw, err := strconv.Unquote(quoted)
	if err != nil {
		t.Fatalf("strconv.Unquote: %v\nquoted=%s", err, quoted)
	}

	var got core.BuildManifest
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("json.Unmarshal round-trip: %v\nraw=%s", err, raw)
	}

	// Re-marshal the original and the round-tripped manifest and compare —
	// json.RawMessage and time.Time do not compare cleanly with reflect.DeepEqual
	// across round trips, but the byte sequences must match.
	a, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal original: %v", err)
	}
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal round-tripped: %v", err)
	}
	if string(a) != string(b) {
		t.Errorf("round-trip mismatch:\noriginal=%s\ngot     =%s", a, b)
	}
}

func TestBuildLabelsEscapesEmbeddedQuotes(t *testing.T) {
	// SBOM contains quotes; the json label value must remain a valid quoted
	// Dockerfile string (each " inside the JSON must be escaped).
	m := fullManifest()
	lines := BuildLabels(m)
	joined := strings.Join(lines, "")

	// strconv.Quote escapes " as \" — that is the exact form Dockerfile
	// expects inside a double-quoted LABEL value.
	if !strings.Contains(joined, `\"bomFormat\":\"CycloneDX\"`) {
		t.Errorf("expected escaped JSON quotes in manifest.json label, got:\n%s", joined)
	}
}

func TestEmitDockerfileWithManifestFinalStageOnly(t *testing.T) {
	plan := &core.BuildPlan{
		Framework: "nextjs",
		Expose:    3000,
		Stages: []core.Stage{
			{
				Name: "build",
				From: "node:22-alpine",
				Steps: []core.Step{
					{Type: core.StepWorkdir, Args: []string{"/app"}},
				},
			},
			{
				Name: "runtime",
				From: "gcr.io/distroless/nodejs22-debian12:nonroot",
				Steps: []core.Step{
					{Type: core.StepWorkdir, Args: []string{"/app"}},
					{Type: core.StepExpose, Args: []string{"3000"}},
					{Type: core.StepCmd, Args: []string{"node", "server.js"}},
				},
			},
		},
	}

	m := fullManifest()
	df := EmitDockerfileWithManifest(plan, &m)
	if df == "" {
		t.Fatalf("empty dockerfile")
	}

	// Split on the build-stage header. Labels must NOT appear before the
	// runtime stage header.
	runtimeStart := strings.Index(df, "FROM gcr.io/distroless/nodejs22-debian12:nonroot AS runtime")
	if runtimeStart < 0 {
		t.Fatalf("runtime stage header missing:\n%s", df)
	}
	beforeRuntime := df[:runtimeStart]
	afterRuntime := df[runtimeStart:]

	if strings.Contains(beforeRuntime, "LABEL io.permanu.") {
		t.Errorf("io.permanu labels leaked into intermediate stage:\n%s", beforeRuntime)
	}
	if !strings.Contains(afterRuntime, `LABEL io.permanu.manifest.schema="1.0"`) {
		t.Errorf("io.permanu.manifest.schema label missing from final stage:\n%s", afterRuntime)
	}
	if !strings.Contains(afterRuntime, `LABEL io.permanu.framework.name="nextjs"`) {
		t.Errorf("io.permanu.framework.name label missing from final stage:\n%s", afterRuntime)
	}
}

func TestEmitDockerfileNilManifestIsBackwardCompatible(t *testing.T) {
	plan := &core.BuildPlan{
		Framework: "go",
		Expose:    8080,
		Stages: []core.Stage{
			{
				Name:  "runtime",
				From:  "gcr.io/distroless/static:nonroot",
				Steps: []core.Step{{Type: core.StepCmd, Args: []string{"/app"}}},
			},
		},
	}
	df := EmitDockerfile(plan)
	if df == "" {
		t.Fatalf("empty dockerfile")
	}
	if strings.Contains(df, "io.permanu.") {
		t.Errorf("EmitDockerfile (nil manifest) leaked io.permanu labels:\n%s", df)
	}
}
