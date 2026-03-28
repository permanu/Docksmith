package docksmith

import (
	"encoding/json"
	"testing"
)

func TestFrameworkJSONRoundTrip(t *testing.T) {
	fw := Framework{
		Name:           "nextjs",
		BuildCommand:   "npm run build",
		StartCommand:   "npm start",
		Port:           3000,
		OutputDir:      ".next",
		NodeVersion:    "22",
		PackageManager: "npm",
		SystemDeps:     []string{"curl", "git"},
	}

	data, err := fw.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	got, err := FrameworkFromJSON(data)
	if err != nil {
		t.Fatalf("FrameworkFromJSON: %v", err)
	}

	if got.Name != fw.Name {
		t.Errorf("Name: got %q, want %q", got.Name, fw.Name)
	}
	if got.BuildCommand != fw.BuildCommand {
		t.Errorf("BuildCommand: got %q, want %q", got.BuildCommand, fw.BuildCommand)
	}
	if got.StartCommand != fw.StartCommand {
		t.Errorf("StartCommand: got %q, want %q", got.StartCommand, fw.StartCommand)
	}
	if got.Port != fw.Port {
		t.Errorf("Port: got %d, want %d", got.Port, fw.Port)
	}
	if got.OutputDir != fw.OutputDir {
		t.Errorf("OutputDir: got %q, want %q", got.OutputDir, fw.OutputDir)
	}
	if got.NodeVersion != fw.NodeVersion {
		t.Errorf("NodeVersion: got %q, want %q", got.NodeVersion, fw.NodeVersion)
	}
	if got.PackageManager != fw.PackageManager {
		t.Errorf("PackageManager: got %q, want %q", got.PackageManager, fw.PackageManager)
	}
	if len(got.SystemDeps) != len(fw.SystemDeps) {
		t.Fatalf("SystemDeps len: got %d, want %d", len(got.SystemDeps), len(fw.SystemDeps))
	}
	for i, dep := range fw.SystemDeps {
		if got.SystemDeps[i] != dep {
			t.Errorf("SystemDeps[%d]: got %q, want %q", i, got.SystemDeps[i], dep)
		}
	}
}

func TestFrameworkOmitemptyFields(t *testing.T) {
	fw := Framework{
		Name:         "go",
		BuildCommand: "go build -o app .",
		StartCommand: "./app",
		Port:         8080,
		GoVersion:    "1.24",
	}

	data, err := fw.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	omitted := []string{
		"output_dir", "node_version", "package_manager",
		"python_version", "python_pm", "system_deps",
		"php_version", "dotnet_version", "java_version",
		"deno_version", "bun_version",
	}
	for _, key := range omitted {
		if _, ok := raw[key]; ok {
			t.Errorf("expected %q to be omitted from JSON, but it was present", key)
		}
	}
}

func TestFrameworkFromJSONEmptyData(t *testing.T) {
	_, err := FrameworkFromJSON(nil)
	if err == nil {
		t.Fatal("expected error for nil data, got nil")
	}

	_, err = FrameworkFromJSON([]byte{})
	if err == nil {
		t.Fatal("expected error for empty data, got nil")
	}
}

func TestFrameworkFromJSONInvalidJSON(t *testing.T) {
	_, err := FrameworkFromJSON([]byte(`{not valid json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestErrorSentinelsDistinct(t *testing.T) {
	if ErrNotDetected == ErrInvalidConfig {
		t.Error("ErrNotDetected and ErrInvalidConfig must be distinct")
	}
	if ErrNotDetected == ErrInvalidPlan {
		t.Error("ErrNotDetected and ErrInvalidPlan must be distinct")
	}
	if ErrInvalidConfig == ErrInvalidPlan {
		t.Error("ErrInvalidConfig and ErrInvalidPlan must be distinct")
	}
}
