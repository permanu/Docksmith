package plan

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/permanu/docksmith/config"
	"github.com/permanu/docksmith/core"
)

// userSpec holds a parsed runtime_config.user value.
type userSpec struct {
	name string // empty means numeric-only
	uid  string // numeric string, may be empty if only name given
	gid  string // numeric string, may be empty
}

// isNumeric reports whether s is a non-empty string of decimal digits.
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.ParseUint(s, 10, 64)
	return err == nil
}

// parseUserSpec splits a user string into (name, uid, gid).
// Accepted formats:
//
//	"name"          → name only (adduser picks uid)
//	"name:uid"      → named user with explicit uid
//	"name:uid:gid"  → named user with explicit uid + gid
//	"uid"           → numeric only, skip creation
//	"uid:gid"       → numeric only, skip creation
//
// Returns (spec, true) when user creation is needed (name is non-numeric).
// Returns (spec, false) when the value is numeric-only — just emit USER.
func parseUserSpec(s string) (userSpec, bool) {
	parts := strings.SplitN(s, ":", 3)
	switch len(parts) {
	case 1:
		if isNumeric(parts[0]) {
			return userSpec{uid: parts[0]}, false
		}
		return userSpec{name: parts[0]}, true
	case 2:
		if isNumeric(parts[0]) {
			// uid:gid — numeric only
			return userSpec{uid: parts[0], gid: parts[1]}, false
		}
		// name:uid
		return userSpec{name: parts[0], uid: parts[1]}, true
	default:
		if isNumeric(parts[0]) {
			return userSpec{uid: parts[0], gid: parts[1]}, false
		}
		// name:uid:gid
		return userSpec{name: parts[0], uid: parts[1], gid: parts[2]}, true
	}
}

// createNamedUser inserts a RUN step into stage that creates the OS user described by spec.
// On distroless images there is no shell or adduser binary; user creation is skipped and
// the caller's USER directive is emitted as-is (the user must already exist in the image,
// e.g. the built-in "nonroot" user, or Docker will error at container start).
func createNamedUser(stage *core.Stage, spec userSpec) error {
	if strings.Contains(stage.From, "distroless") {
		// No shell available — cannot run adduser/useradd. Just emit USER below.
		return nil
	}

	group := spec.name
	var cmd string
	if strings.Contains(stage.From, "alpine") {
		// BusyBox addgroup/adduser
		groupCmd := fmt.Sprintf("addgroup -S %s", group)
		if spec.gid != "" {
			groupCmd = fmt.Sprintf("addgroup -S -g %s %s", spec.gid, group)
		}
		userCmd := fmt.Sprintf("adduser -S -G %s %s", group, spec.name)
		if spec.uid != "" {
			userCmd = fmt.Sprintf("adduser -S -G %s -u %s %s", group, spec.uid, spec.name)
		}
		cmd = groupCmd + " && " + userCmd
	} else {
		// Debian/Ubuntu shadow-utils groupadd/useradd
		groupCmd := fmt.Sprintf("groupadd --system %s", group)
		if spec.gid != "" {
			groupCmd = fmt.Sprintf("groupadd --system -g %s %s", spec.gid, group)
		}
		userCmd := fmt.Sprintf("useradd --system --no-create-home --gid %s %s", group, spec.name)
		if spec.uid != "" {
			userCmd = fmt.Sprintf("useradd --system --no-create-home -u %s --gid %s %s", spec.uid, group, spec.name)
		}
		cmd = groupCmd + " && " + userCmd
	}

	stage.Steps = append(stage.Steps, core.Step{Type: core.StepRun, Args: []string{cmd}})
	return nil
}

// addNonRootUser appends user setup steps to a stage.
// When builtInUser is non-empty (e.g. "node", "nginx"), the image already has
// that user — just switch to it. When empty, create appgroup + appuser first.
func addNonRootUser(stage *core.Stage, builtInUser string) {
	if builtInUser != "" {
		stage.Steps = append(stage.Steps, core.Step{
			Type: core.StepUser,
			Args: []string{builtInUser},
		})
		return
	}
	// Alpine uses addgroup/adduser (BusyBox), Debian uses groupadd/useradd (shadow-utils).
	// Detect by checking if the stage FROM image contains "alpine".
	var createCmd string
	if strings.Contains(stage.From, "alpine") {
		createCmd = "addgroup -S appgroup && adduser -S -G appgroup appuser"
	} else {
		createCmd = "groupadd --system appgroup && useradd --system --no-create-home --gid appgroup appuser"
	}
	stage.Steps = append(stage.Steps,
		core.Step{Type: core.StepRun, Args: []string{createCmd}},
		core.Step{Type: core.StepUser, Args: []string{"appuser"}},
	)
}

// addHealthcheck appends a HEALTHCHECK step appropriate for the runtime.
// Go and Rust use distroless images with no shell — no healthcheck is added.
// opts may be nil; when nil the hardcoded defaults are rendered at emit time.
func addHealthcheck(stage *core.Stage, runtime string, port int, opts *config.HealthcheckOpts) {
	cmd := healthcheckCmd(runtime, port)
	if cmd == "" {
		return
	}
	step := core.Step{
		Type: core.StepHealthcheck,
		Args: []string{cmd},
	}
	if opts != nil {
		o := core.HealthcheckOpts{
			Interval:    opts.Interval,
			Timeout:     opts.Timeout,
			StartPeriod: opts.StartPeriod,
			Retries:     opts.Retries,
		}
		step.HealthcheckOpts = &o
	}
	stage.Steps = append(stage.Steps, step)
}

func healthcheckCmd(runtime string, port int) string {
	switch runtime {
	case "go", "rust":
		// Distroless: no shell, no curl, no wget.
		return ""
	case "node":
		return fmt.Sprintf(
			`node -e "const http=require('http');http.get('http://localhost:%d/',r=>{process.exit(r.statusCode===200?0:1)}).on('error',()=>process.exit(1))"`,
			port,
		)
	case "python":
		return fmt.Sprintf(
			`python -c "import urllib.request; urllib.request.urlopen('http://localhost:%d/')"`,
			port,
		)
	case "ruby":
		return fmt.Sprintf(
			`ruby -e "require 'net/http'; Net::HTTP.get(URI('http://localhost:%d/'))"`,
			port,
		)
	case "java":
		// Alpine JRE images don't have curl; wget is available by default.
		return fmt.Sprintf("wget -q --spider http://localhost:%d/", port)
	case "php", "dotnet":
		return fmt.Sprintf("curl -f http://localhost:%d/", port)
	case "elixir":
		return fmt.Sprintf("wget -q --spider http://localhost:%d/", port)
	case "bun":
		return fmt.Sprintf(
			`bun -e "fetch('http://localhost:%d/').then(r=>{if(!r.ok)process.exit(1)}).catch(()=>process.exit(1))"`,
			port,
		)
	case "deno":
		return fmt.Sprintf(
			`deno eval "const r=await fetch('http://localhost:%d/');if(!r.ok)Deno.exit(1)"`,
			port,
		)
	case "static":
		return "curl -f http://localhost:80/"
	default:
		return fmt.Sprintf("curl -f http://localhost:%d/", port)
	}
}

// addTini installs tini in the builder and wires it as the runtime ENTRYPOINT.
// tini reaps zombie processes and forwards signals — critical for Node/Python workloads.
func addTini(builder, runtime *core.Stage) {
	var installCmd string
	var tiniPath string
	if strings.Contains(builder.From, "alpine") {
		installCmd = "apk add --no-cache tini"
		tiniPath = "/sbin/tini" // Alpine installs tini to /sbin/
	} else {
		installCmd = withAptCleanup("apt-get update -qq && apt-get install -y --no-install-recommends tini")
		tiniPath = "/usr/bin/tini"
	}
	builder.Steps = append(builder.Steps, core.Step{
		Type: core.StepRun,
		Args: []string{installCmd},
	})
	runtime.Steps = append(runtime.Steps,
		core.Step{
			Type:     core.StepCopyFrom,
			CopyFrom: &core.CopyFrom{Stage: builder.Name, Src: tiniPath, Dst: tiniPath},
		},
		core.Step{Type: core.StepEntrypoint, Args: []string{tiniPath, "--"}},
	)
}

// withAptCleanup appends apt list cleanup to a shell command.
func withAptCleanup(cmd string) string {
	if strings.Contains(cmd, "rm -rf /var/lib/apt/lists/*") {
		return cmd
	}
	return cmd + " && rm -rf /var/lib/apt/lists/*"
}
