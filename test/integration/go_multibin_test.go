package integration_test

import (
	"strings"
	"testing"

	"github.com/permanu/docksmith"
	"github.com/permanu/docksmith/config"
)

func TestGo_MultiBinary_TwoRUNBuildSteps(t *testing.T) {
	bins := []config.Binary{
		{Name: "server", Path: "./cmd/server"},
		{Name: "worker", Path: "./cmd/worker"},
	}
	fw := &docksmith.Framework{
		Name:         "go-std",
		GoVersion:    "1.22",
		Port:         8080,
		StartCommand: "/usr/local/bin/server",
	}
	df, err := docksmith.GenerateDockerfile(fw, docksmith.WithBinaries(bins))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expect two separate RUN go build steps.
	serverBuild := `CGO_ENABLED=0 go build -ldflags="-w -s" -o /out/server ./cmd/server`
	workerBuild := `CGO_ENABLED=0 go build -ldflags="-w -s" -o /out/worker ./cmd/worker`
	if !strings.Contains(df, serverBuild) {
		t.Errorf("Dockerfile missing server build step\ngot:\n%s", df)
	}
	if !strings.Contains(df, workerBuild) {
		t.Errorf("Dockerfile missing worker build step\ngot:\n%s", df)
	}

	// Expect two COPY --from=builder steps in runtime stage.
	serverCopy := `COPY --from=builder /out/server /usr/local/bin/server`
	workerCopy := `COPY --from=builder /out/worker /usr/local/bin/worker`
	if !strings.Contains(df, serverCopy) {
		t.Errorf("Dockerfile missing server COPY step\ngot:\n%s", df)
	}
	if !strings.Contains(df, workerCopy) {
		t.Errorf("Dockerfile missing worker COPY step\ngot:\n%s", df)
	}

	// CMD must default to first binary (server).
	if !strings.Contains(df, `CMD ["/usr/local/bin/server"]`) {
		t.Errorf("CMD should default to first binary\ngot:\n%s", df)
	}
}

func TestGo_MultiBinary_OutputNameOverride(t *testing.T) {
	bins := []config.Binary{
		{Name: "server", Path: "./cmd/server", OutputName: "api"},
		{Name: "worker", Path: "./cmd/worker"},
	}
	fw := &docksmith.Framework{
		Name:         "go-std",
		GoVersion:    "1.22",
		Port:         8080,
		StartCommand: "/usr/local/bin/api",
	}
	df, err := docksmith.GenerateDockerfile(fw, docksmith.WithBinaries(bins))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(df, `-o /out/api ./cmd/server`) {
		t.Errorf("expected output_name 'api' used in build step\ngot:\n%s", df)
	}
	if !strings.Contains(df, `COPY --from=builder /out/api /usr/local/bin/api`) {
		t.Errorf("expected output_name 'api' in COPY step\ngot:\n%s", df)
	}
	// CMD defaults to first binary output name.
	if !strings.Contains(df, `CMD ["/usr/local/bin/api"]`) {
		t.Errorf("CMD should use output_name of first binary\ngot:\n%s", df)
	}
}

func TestGo_MultiBinary_StartCommandOverride(t *testing.T) {
	bins := []config.Binary{
		{Name: "server", Path: "./cmd/server"},
		{Name: "worker", Path: "./cmd/worker"},
	}
	fw := &docksmith.Framework{
		Name:         "go-std",
		GoVersion:    "1.22",
		Port:         8080,
		StartCommand: "/usr/local/bin/server",
	}
	customCmd := "/usr/local/bin/worker --flag"
	df, err := docksmith.GenerateDockerfile(fw, docksmith.WithBinaries(bins), docksmith.WithStartCommand(customCmd))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(df, `CMD ["/usr/local/bin/worker", "--flag"]`) {
		t.Errorf("Start.Command override should win over default CMD\ngot:\n%s", df)
	}
}

func TestGo_MultiBinary_LdFlagsAppliedToAllBinaries(t *testing.T) {
	bins := []config.Binary{
		{Name: "server", Path: "./cmd/server"},
		{Name: "worker", Path: "./cmd/worker"},
	}
	fw := &docksmith.Framework{
		Name:         "go-std",
		GoVersion:    "1.22",
		Port:         8080,
		StartCommand: "/usr/local/bin/server",
	}
	df, err := docksmith.GenerateDockerfile(fw,
		docksmith.WithBinaries(bins),
		docksmith.WithLdFlags(map[string]string{"main.Version": "1.0.0"}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantFlag := `-X main.Version=1.0.0`
	// Count occurrences — should appear in both build steps.
	count := strings.Count(df, wantFlag)
	if count != 2 {
		t.Errorf("ldflags should appear in both binary build steps, found %d occurrences\ngot:\n%s", count, df)
	}
}

func TestGo_MultiBinary_Fixture(t *testing.T) {
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

	if !strings.Contains(df, `-o /out/server ./cmd/server`) {
		t.Errorf("Dockerfile missing server build step\ngot:\n%s", df)
	}
	if !strings.Contains(df, `-o /out/worker ./cmd/worker`) {
		t.Errorf("Dockerfile missing worker build step\ngot:\n%s", df)
	}
	if !strings.Contains(df, `COPY --from=builder /out/server /usr/local/bin/server`) {
		t.Errorf("Dockerfile missing server COPY step\ngot:\n%s", df)
	}
	if !strings.Contains(df, `COPY --from=builder /out/worker /usr/local/bin/worker`) {
		t.Errorf("Dockerfile missing worker COPY step\ngot:\n%s", df)
	}
}
