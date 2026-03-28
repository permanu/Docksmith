package plan

import (
	"github.com/permanu/docksmith/core"
	"strconv"
	"strings"
)

// planRuby builds a two-stage BuildPlan for Ruby applications (Rails, Sinatra, plain Ruby).
// Stage "builder" installs system deps and bundles gems.
// Stage "runtime" copies the bundled app and launches the server.
func planRuby(fw *core.Framework) (*core.BuildPlan, error) {
	// No dedicated RubyVersion field in Framework; pass "" to get the default.
	image := ResolveDockerTag("ruby", "")

	startCmd := fw.StartCommand
	if startCmd == "" {
		startCmd = "bundle exec ruby app.rb"
	}
	port := fw.Port
	if port == 0 {
		port = 3000
	}

	startArgs := strings.Fields(startCmd)

	builder := core.Stage{
		Name: "builder",
		From: image,
		Steps: []core.Step{
			{
				Type: core.StepRun,
				Args: []string{
					"apt-get update -qq && apt-get install -y --no-install-recommends " +
						"build-essential libpq-dev libyaml-dev libffi-dev && " +
						"rm -rf /var/lib/apt/lists/*",
				},
			},
			{Type: core.StepWorkdir, Args: []string{"/app"}},
			{Type: core.StepCopy, Args: []string{"Gemfile", "Gemfile.lock*", "./"}},
			{
				Type: core.StepRun,
				Args: []string{
					"bundle config set --local without 'development test' && " +
						"bundle install --jobs 4 --retry 3",
				},
			},
			{Type: core.StepCopy, Args: []string{".", "."}},
		},
	}

	if fw.Name == "rails" {
		builder.Steps = append(builder.Steps, core.Step{
			Type: core.StepRun,
			Args: []string{"bundle exec rake assets:precompile 2>/dev/null || true"},
		})
	}

	runtime := core.Stage{
		Name: "runtime",
		From: image,
		Steps: []core.Step{
			{
				Type: core.StepRun,
				Args: []string{
					"apt-get update -qq && apt-get install -y --no-install-recommends libpq5 && " +
						"rm -rf /var/lib/apt/lists/*",
				},
			},
			{Type: core.StepWorkdir, Args: []string{"/app"}},
			{
				Type:     core.StepCopyFrom,
				CopyFrom: &core.CopyFrom{Stage: "builder", Src: "/usr/local/bundle", Dst: "/usr/local/bundle"},
				Link:     true,
			},
			{
				Type:     core.StepCopyFrom,
				CopyFrom: &core.CopyFrom{Stage: "builder", Src: "/app", Dst: "."},
				Link:     true,
			},
			{Type: core.StepEnv, Args: []string{"PATH", "/usr/local/bundle/bin:$PATH"}},
			{Type: core.StepEnv, Args: []string{"RAILS_ENV", "production"}},
			{Type: core.StepEnv, Args: []string{"RAILS_LOG_TO_STDOUT", "true"}},
			{Type: core.StepExpose, Args: []string{strconv.Itoa(port)}},
			{Type: core.StepCmd, Args: startArgs},
		},
	}

	addNonRootUser(&runtime, "")
	addHealthcheck(&runtime, "ruby", port)

	return &core.BuildPlan{
		Framework:    fw.Name,
		Stages:       []core.Stage{builder, runtime},
		Expose:       port,
		Dockerignore: []string{".git", "tmp", "log", ".bundle"},
	}, nil
}
