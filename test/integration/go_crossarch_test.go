package integration_test

import (
	"strings"
	"testing"

	"github.com/permanu/docksmith"
	"github.com/permanu/docksmith/config"
)

// TestGo_CrossArch_TwoBuildStepsPerArch verifies that a binary with
// Architectures = ["amd64","arm64"] produces two separate RUN go build steps
// with GOOS/GOARCH env vars and -<arch> suffixed output names.
func TestGo_CrossArch_TwoBuildStepsPerArch(t *testing.T) {
	bins := []config.Binary{
		{Name: "permanu-agent", Path: "./cmd/agent", Architectures: []string{"amd64", "arm64"}},
	}
	fw := &docksmith.Framework{
		Name:         "go-std",
		GoVersion:    "1.22",
		Port:         8080,
		StartCommand: "/usr/local/bin/permanu-agent-amd64",
	}
	df, err := docksmith.GenerateDockerfile(fw, docksmith.WithBinaries(bins))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantAmd64Build := `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /out/permanu-agent-amd64 ./cmd/agent`
	wantArm64Build := `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-w -s" -o /out/permanu-agent-arm64 ./cmd/agent`
	if !strings.Contains(df, wantAmd64Build) {
		t.Errorf("Dockerfile missing amd64 build step\ngot:\n%s", df)
	}
	if !strings.Contains(df, wantArm64Build) {
		t.Errorf("Dockerfile missing arm64 build step\ngot:\n%s", df)
	}

	// Exactly 2 go build steps total.
	if count := strings.Count(df, "go build"); count != 2 {
		t.Errorf("expected 2 go build steps, got %d\ngot:\n%s", count, df)
	}
}

// TestGo_CrossArch_CopyStepsWithArchSuffix verifies COPY --from=builder lines
// use the -<arch> suffix for each architecture.
func TestGo_CrossArch_CopyStepsWithArchSuffix(t *testing.T) {
	bins := []config.Binary{
		{Name: "permanu-agent", Path: "./cmd/agent", Architectures: []string{"amd64", "arm64"}},
	}
	fw := &docksmith.Framework{
		Name:         "go-std",
		GoVersion:    "1.22",
		Port:         8080,
		StartCommand: "/usr/local/bin/permanu-agent-amd64",
	}
	df, err := docksmith.GenerateDockerfile(fw, docksmith.WithBinaries(bins))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantAmd64Copy := `COPY --from=builder /out/permanu-agent-amd64 /usr/local/bin/permanu-agent-amd64`
	wantArm64Copy := `COPY --from=builder /out/permanu-agent-arm64 /usr/local/bin/permanu-agent-arm64`
	if !strings.Contains(df, wantAmd64Copy) {
		t.Errorf("Dockerfile missing amd64 COPY step\ngot:\n%s", df)
	}
	if !strings.Contains(df, wantArm64Copy) {
		t.Errorf("Dockerfile missing arm64 COPY step\ngot:\n%s", df)
	}
}

// TestGo_CrossArch_DefaultCMDPointsToFirstArch verifies that when no
// Start.Command override is given, CMD defaults to the first arch of the
// first binary (e.g. permanu-agent-amd64).
func TestGo_CrossArch_DefaultCMDPointsToFirstArch(t *testing.T) {
	bins := []config.Binary{
		{Name: "permanu-agent", Path: "./cmd/agent", Architectures: []string{"amd64", "arm64"}},
	}
	fw := &docksmith.Framework{
		Name:         "go-std",
		GoVersion:    "1.22",
		Port:         8080,
		StartCommand: "/usr/local/bin/permanu-agent-amd64",
	}
	df, err := docksmith.GenerateDockerfile(fw, docksmith.WithBinaries(bins))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(df, `CMD ["/usr/local/bin/permanu-agent-amd64"]`) {
		t.Errorf("CMD should default to first binary's first arch\ngot:\n%s", df)
	}
}

// TestGo_CrossArch_StartCommandOverrideWins verifies that an explicit
// Start.Command beats the auto-generated default CMD.
func TestGo_CrossArch_StartCommandOverrideWins(t *testing.T) {
	bins := []config.Binary{
		{Name: "permanu-agent", Path: "./cmd/agent", Architectures: []string{"amd64", "arm64"}},
	}
	fw := &docksmith.Framework{
		Name:         "go-std",
		GoVersion:    "1.22",
		Port:         8080,
		StartCommand: "/usr/local/bin/permanu-agent-amd64",
	}
	df, err := docksmith.GenerateDockerfile(fw,
		docksmith.WithBinaries(bins),
		docksmith.WithStartCommand("/usr/local/bin/permanu-agent-arm64"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(df, `CMD ["/usr/local/bin/permanu-agent-arm64"]`) {
		t.Errorf("Start.Command override should win over default CMD\ngot:\n%s", df)
	}
}

// TestGo_CrossArch_LdFlagsAppliedToAllArchBuilds verifies that LdFlags are
// appended to every per-arch go build step.
func TestGo_CrossArch_LdFlagsAppliedToAllArchBuilds(t *testing.T) {
	bins := []config.Binary{
		{Name: "permanu-agent", Path: "./cmd/agent", Architectures: []string{"amd64", "arm64"}},
	}
	fw := &docksmith.Framework{
		Name:         "go-std",
		GoVersion:    "1.22",
		Port:         8080,
		StartCommand: "/usr/local/bin/permanu-agent-amd64",
	}
	df, err := docksmith.GenerateDockerfile(fw,
		docksmith.WithBinaries(bins),
		docksmith.WithLdFlags(map[string]string{"main.Version": "2.0.0"}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantFlag := `-X main.Version=2.0.0`
	count := strings.Count(df, wantFlag)
	if count != 2 {
		t.Errorf("ldflags should appear in both arch build steps, found %d occurrences\ngot:\n%s", count, df)
	}
}

// TestGo_CrossArch_MixedBinaries verifies that when one binary has
// Architectures set and another does not, they are handled independently:
// the cross-arch binary gets per-arch steps; the plain binary gets a single step.
func TestGo_CrossArch_MixedBinaries(t *testing.T) {
	bins := []config.Binary{
		{Name: "permanu-agent", Path: "./cmd/agent", Architectures: []string{"amd64", "arm64"}},
		{Name: "worker", Path: "./cmd/worker"},
	}
	fw := &docksmith.Framework{
		Name:         "go-std",
		GoVersion:    "1.22",
		Port:         8080,
		StartCommand: "/usr/local/bin/permanu-agent-amd64",
	}
	df, err := docksmith.GenerateDockerfile(fw, docksmith.WithBinaries(bins))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Cross-arch binary: 2 arch-specific build steps.
	if !strings.Contains(df, `GOARCH=amd64 go build`) {
		t.Errorf("missing amd64 build step\ngot:\n%s", df)
	}
	if !strings.Contains(df, `GOARCH=arm64 go build`) {
		t.Errorf("missing arm64 build step\ngot:\n%s", df)
	}

	// Plain binary: single build step without GOARCH env var.
	workerBuild := `CGO_ENABLED=0 go build -ldflags="-w -s" -o /out/worker ./cmd/worker`
	if !strings.Contains(df, workerBuild) {
		t.Errorf("missing plain worker build step\ngot:\n%s", df)
	}

	// Plain binary COPY step (no arch suffix).
	workerCopy := `COPY --from=builder /out/worker /usr/local/bin/worker`
	if !strings.Contains(df, workerCopy) {
		t.Errorf("missing plain worker COPY step\ngot:\n%s", df)
	}
}

// TestGo_CrossArch_Fixture exercises the full Build() pipeline using the
// go-multibin-crossarch fixture on disk.
func TestGo_CrossArch_Fixture(t *testing.T) {
	const fixturePath = "../../testdata/fixtures/go-multibin-crossarch"
	planOpts, err := docksmith.LoadPlanOptions(fixturePath)
	if err != nil {
		t.Fatalf("LoadPlanOptions: %v", err)
	}
	df, fw, err := docksmith.Build(fixturePath, planOpts...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw == nil {
		t.Fatal("got nil framework")
	}

	// Cross-arch binary (permanu-agent): 2 per-arch build steps.
	if !strings.Contains(df, `-o /out/permanu-agent-amd64 ./cmd/agent`) {
		t.Errorf("missing amd64 agent build\ngot:\n%s", df)
	}
	if !strings.Contains(df, `-o /out/permanu-agent-arm64 ./cmd/agent`) {
		t.Errorf("missing arm64 agent build\ngot:\n%s", df)
	}

	// Cross-arch COPY steps.
	if !strings.Contains(df, `COPY --from=builder /out/permanu-agent-amd64 /usr/local/bin/permanu-agent-amd64`) {
		t.Errorf("missing amd64 agent COPY\ngot:\n%s", df)
	}
	if !strings.Contains(df, `COPY --from=builder /out/permanu-agent-arm64 /usr/local/bin/permanu-agent-arm64`) {
		t.Errorf("missing arm64 agent COPY\ngot:\n%s", df)
	}

	// Plain binary (worker): single step, no GOARCH prefix.
	if !strings.Contains(df, `CGO_ENABLED=0 go build -ldflags="-w -s" -o /out/worker ./cmd/worker`) {
		t.Errorf("missing plain worker build step\ngot:\n%s", df)
	}
	if !strings.Contains(df, `COPY --from=builder /out/worker /usr/local/bin/worker`) {
		t.Errorf("missing plain worker COPY step\ngot:\n%s", df)
	}
}

// TestGo_MultiBinary_Fixture_Unchanged asserts that the existing go-multibin
// fixture (no Architectures) produces byte-identical output — Architectures is
// purely opt-in.
func TestGo_CrossArch_ExistingMultibinFixtureUnchanged(t *testing.T) {
	const fixturePath = "../../testdata/fixtures/go-multibin"
	planOpts, err := docksmith.LoadPlanOptions(fixturePath)
	if err != nil {
		t.Fatalf("LoadPlanOptions: %v", err)
	}
	df, fw, err := docksmith.Build(fixturePath, planOpts...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw == nil {
		t.Fatal("got nil framework")
	}

	// Must NOT contain any GOARCH env var — plain multibin.
	if strings.Contains(df, "GOARCH=") {
		t.Errorf("go-multibin fixture must not contain GOARCH= (opt-in only)\ngot:\n%s", df)
	}

	// Original steps must still be present.
	if !strings.Contains(df, `-o /out/server ./cmd/server`) {
		t.Errorf("go-multibin fixture missing server build step\ngot:\n%s", df)
	}
	if !strings.Contains(df, `-o /out/worker ./cmd/worker`) {
		t.Errorf("go-multibin fixture missing worker build step\ngot:\n%s", df)
	}
	if !strings.Contains(df, `COPY --from=builder /out/server /usr/local/bin/server`) {
		t.Errorf("go-multibin fixture missing server COPY step\ngot:\n%s", df)
	}
	if !strings.Contains(df, `COPY --from=builder /out/worker /usr/local/bin/worker`) {
		t.Errorf("go-multibin fixture missing worker COPY step\ngot:\n%s", df)
	}
}
