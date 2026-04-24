package docksmith

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// installTestTP installs an in-memory TracerProvider and returns the exporter
// and a cleanup function. The cleanup restores the previous global provider.
func installTestTP(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	prev := otel.GetTracerProvider()
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(prev)
	})
	return exp
}

// goProjectDir creates a minimal Go project in a temp dir for detection.
func goProjectDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testapp\n\ngo 1.26\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644))
	return dir
}

// spanNames extracts span names from the recorded spans.
func spanNames(spans tracetest.SpanStubs) []string {
	names := make([]string, len(spans))
	for i, s := range spans {
		names[i] = s.Name
	}
	return names
}

// hasSpan returns true if any recorded span matches name.
func hasSpan(spans tracetest.SpanStubs, name string) bool {
	for _, s := range spans {
		if s.Name == name {
			return true
		}
	}
	return false
}

// TestBuildWithOptions_EmitsClassifyAndPlanSpans verifies that BuildWithOptions
// records docksmith.build, docksmith.classify, and docksmith.plan spans.
func TestBuildWithOptions_EmitsClassifyAndPlanSpans(t *testing.T) {
	exp := installTestTP(t)
	dir := goProjectDir(t)

	_, _, err := BuildWithOptions(dir, DetectOptions{})
	if err != nil {
		t.Fatalf("BuildWithOptions: %v", err)
	}

	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected spans, got none")
	}
	for _, want := range []string{"docksmith.build", "docksmith.classify", "docksmith.plan"} {
		if !hasSpan(spans, want) {
			t.Errorf("missing span %q; recorded: %v", want, spanNames(spans))
		}
	}
}

// TestBuildWithManifest_EmitsSpans verifies that BuildWithManifest records the
// expected per-stage spans when a TracerProvider is installed.
func TestBuildWithManifest_EmitsSpans(t *testing.T) {
	exp := installTestTP(t)
	dir := goProjectDir(t)

	extras := ManifestExtras{
		BuildID:         "018f2b1a-0000-7c2b-0000-4c1f2d3e4f99",
		Commit:          "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		BaseImageDigest: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
	}

	_, _, err := BuildWithManifest(context.Background(), dir, extras, DetectOptions{})
	if err != nil {
		t.Fatalf("BuildWithManifest: %v", err)
	}

	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected spans, got none")
	}
	for _, want := range []string{"docksmith.build", "docksmith.classify", "docksmith.plan"} {
		if !hasSpan(spans, want) {
			t.Errorf("missing span %q; recorded: %v", want, spanNames(spans))
		}
	}
}

// TestBuildSpan_ClassifyAttributes verifies docksmith.classify sets the
// detected_language attribute.
func TestBuildSpan_ClassifyAttributes(t *testing.T) {
	exp := installTestTP(t)
	dir := goProjectDir(t)

	_, _, err := BuildWithOptions(dir, DetectOptions{})
	if err != nil {
		t.Fatalf("BuildWithOptions: %v", err)
	}

	spans := exp.GetSpans()
	for _, s := range spans {
		if s.Name != "docksmith.classify" {
			continue
		}
		for _, kv := range s.Attributes {
			if string(kv.Key) == "docksmith.detected_language" && kv.Value.AsString() != "" {
				return // pass
			}
		}
		t.Error("docksmith.classify span missing docksmith.detected_language attribute")
		return
	}
	t.Error("docksmith.classify span not found")
}

// TestBuildSpan_NoProviderIsNoop verifies that calling BuildWithOptions without
// any TracerProvider installed completes without error (no-op tracer).
func TestBuildSpan_NoProviderIsNoop(t *testing.T) {
	// Ensure the global provider is the default no-op.
	otel.SetTracerProvider(otel.GetTracerProvider())

	dir := goProjectDir(t)
	_, _, err := BuildWithOptions(dir, DetectOptions{})
	if err != nil {
		t.Fatalf("BuildWithOptions with no-op tracer: %v", err)
	}
}

// TestBuildSpan_ParentContextPropagation verifies child spans share the parent
// trace ID when a parent span is passed via context.
func TestBuildSpan_ParentContextPropagation(t *testing.T) {
	exp := installTestTP(t)
	dir := goProjectDir(t)

	parentCtx, parentSpan := tracer().Start(context.Background(), "test.parent")
	wantTraceID := parentSpan.SpanContext().TraceID()

	_, _, err := BuildWithManifest(parentCtx, dir, ManifestExtras{
		BuildID:         "018f0000-0000-0000-0000-000000000001",
		Commit:          "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		BaseImageDigest: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
	}, DetectOptions{})
	parentSpan.End()
	if err != nil {
		t.Fatalf("BuildWithManifest: %v", err)
	}

	for _, s := range exp.GetSpans() {
		if s.Name == "docksmith.build" {
			if s.SpanContext.TraceID() != wantTraceID {
				t.Errorf("docksmith.build TraceID = %v, want %v", s.SpanContext.TraceID(), wantTraceID)
			}
			return
		}
	}
	t.Error("docksmith.build span not found")
}
