package detect

import (
	"github.com/permanu/docksmith/core"
	"path/filepath"
)

func init() {
	RegisterDetector("elixir-phoenix", detectElixirPhoenix)
}

func detectElixirPhoenix(dir string) *core.Framework {
	if !hasFile(dir, "mix.exs") || !fileContains(filepath.Join(dir, "mix.exs"), "phoenix") {
		return nil
	}
	return &core.Framework{
		Name:         "elixir-phoenix",
		BuildCommand: "mix deps.get && mix compile",
		StartCommand: "mix phx.server",
		Port:         4000,
	}
}
