package docksmith

import "path/filepath"

func init() {
	RegisterDetector("elixir-phoenix", detectElixirPhoenix)
}

func detectElixirPhoenix(dir string) *Framework {
	if !hasFile(dir, "mix.exs") || !fileContains(filepath.Join(dir, "mix.exs"), "phoenix") {
		return nil
	}
	return &Framework{
		Name:         "elixir-phoenix",
		BuildCommand: "mix deps.get && mix compile",
		StartCommand: "mix phx.server",
		Port:         4000,
	}
}
