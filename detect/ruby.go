package detect

import (
	"github.com/permanu/docksmith/core"
	"path/filepath"
)

func init() {
	// Rails before Sinatra — a Rails Gemfile could also match Sinatra heuristics.
	RegisterDetector("sinatra", detectSinatra)
	RegisterDetector("rails", detectRails)
}

func detectRails(dir string) *core.Framework {
	if hasFile(dir, "Gemfile") && hasFile(dir, "config/routes.rb") {
		return &core.Framework{
			Name:         "rails",
			BuildCommand: "bundle install",
			StartCommand: "rails server -b 0.0.0.0 -p 3000",
			Port:         3000,
		}
	}
	return nil
}

func detectSinatra(dir string) *core.Framework {
	if hasFile(dir, "Gemfile") && fileContains(filepath.Join(dir, "Gemfile"), "sinatra") {
		return &core.Framework{
			Name:         "sinatra",
			BuildCommand: "bundle install",
			StartCommand: "ruby app.rb -o 0.0.0.0 -p 4567",
			Port:         4567,
		}
	}
	return nil
}
