package docksmith

import (
	"strconv"
	"strings"
)

// planRuby builds a two-stage BuildPlan for Ruby applications (Rails, Sinatra, plain Ruby).
// Stage "builder" installs system deps and bundles gems.
// Stage "runtime" copies the bundled app and launches the server.
func planRuby(fw *Framework) (*BuildPlan, error) {
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

	builder := Stage{
		Name: "builder",
		From: image,
		Steps: []Step{
			{
				Type: StepRun,
				Args: []string{
					"apt-get update -qq && apt-get install -y --no-install-recommends " +
						"build-essential libpq-dev libyaml-dev libffi-dev && " +
						"rm -rf /var/lib/apt/lists/*",
				},
			},
			{Type: StepWorkdir, Args: []string{"/app"}},
			{Type: StepCopy, Args: []string{"Gemfile", "Gemfile.lock*", "./"}},
			{
				Type: StepRun,
				Args: []string{
					"bundle config set --local without 'development test' && " +
						"bundle install --jobs 4 --retry 3",
				},
				CacheMount: &CacheMount{Target: "/usr/local/bundle"},
			},
			{Type: StepCopy, Args: []string{".", "."}},
		},
	}

	if fw.Name == "rails" {
		builder.Steps = append(builder.Steps, Step{
			Type: StepRun,
			Args: []string{"bundle exec rake assets:precompile 2>/dev/null || true"},
		})
	}

	runtime := Stage{
		Name: "runtime",
		From: image,
		Steps: []Step{
			{
				Type: StepRun,
				Args: []string{
					"apt-get update -qq && apt-get install -y --no-install-recommends libpq5 && " +
						"rm -rf /var/lib/apt/lists/*",
				},
			},
			{Type: StepWorkdir, Args: []string{"/app"}},
			{
				Type:     StepCopyFrom,
				CopyFrom: &CopyFrom{Stage: "builder", Src: "/usr/local/bundle", Dst: "/usr/local/bundle"},
				Link:     true,
			},
			{
				Type:     StepCopyFrom,
				CopyFrom: &CopyFrom{Stage: "builder", Src: "/app", Dst: "."},
				Link:     true,
			},
			{Type: StepEnv, Args: []string{"RAILS_ENV", "production"}},
			{Type: StepEnv, Args: []string{"RAILS_LOG_TO_STDOUT", "true"}},
			{Type: StepExpose, Args: []string{strconv.Itoa(port)}},
			{Type: StepCmd, Args: startArgs},
		},
	}

	return &BuildPlan{
		Framework:    fw.Name,
		Stages:       []Stage{builder, runtime},
		Expose:       port,
		Dockerignore: []string{".git", "tmp", "log", ".bundle"},
	}, nil
}
