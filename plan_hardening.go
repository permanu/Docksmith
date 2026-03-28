package docksmith

import (
	"fmt"
	"strings"
)

// addNonRootUser appends user setup steps to a stage.
// When builtInUser is non-empty (e.g. "node", "nginx"), the image already has
// that user — just switch to it. When empty, create appgroup + appuser first.
func addNonRootUser(stage *Stage, builtInUser string) {
	if builtInUser != "" {
		stage.Steps = append(stage.Steps, Step{
			Type: StepUser,
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
		Step{Type: StepRun, Args: []string{createCmd}},
		Step{Type: StepUser, Args: []string{"appuser"}},
	)
}

// addHealthcheck appends a HEALTHCHECK step appropriate for the runtime.
// Go and Rust use distroless images with no shell — no healthcheck is added.
func addHealthcheck(stage *Stage, runtime string, port int) {
	cmd := healthcheckCmd(runtime, port)
	if cmd == "" {
		return
	}
	stage.Steps = append(stage.Steps, Step{
		Type: StepHealthcheck,
		Args: []string{cmd},
	})
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
func addTini(builder, runtime *Stage) {
	var installCmd string
	var tiniPath string
	if strings.Contains(builder.From, "alpine") {
		installCmd = "apk add --no-cache tini"
		tiniPath = "/sbin/tini" // Alpine installs tini to /sbin/
	} else {
		installCmd = withAptCleanup("apt-get update -qq && apt-get install -y --no-install-recommends tini")
		tiniPath = "/usr/bin/tini"
	}
	builder.Steps = append(builder.Steps, Step{
		Type: StepRun,
		Args: []string{installCmd},
	})
	runtime.Steps = append(runtime.Steps,
		Step{
			Type:     StepCopyFrom,
			CopyFrom: &CopyFrom{Stage: builder.Name, Src: tiniPath, Dst: tiniPath},
		},
		Step{Type: StepEntrypoint, Args: []string{tiniPath, "--"}},
	)
}

// withAptCleanup appends apt list cleanup to a shell command.
func withAptCleanup(cmd string) string {
	if strings.Contains(cmd, "rm -rf /var/lib/apt/lists/*") {
		return cmd
	}
	return cmd + " && rm -rf /var/lib/apt/lists/*"
}
