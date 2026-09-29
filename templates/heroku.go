// SPDX-License-Identifier: MPL-2.0

package templates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/permanu/docksmith/blueprint"
)

type herokuApp struct {
	Name      string                `json:"name"`
	Env       map[string]herokuEnv  `json:"env"`
	Formation map[string]herokuProc `json:"formation"`
	Addons    []herokuAddon         `json:"addons"`
	Scripts   herokuScripts         `json:"scripts"`
	Image     string                `json:"image"`
}

type herokuEnv struct {
	Value     string `json:"value"`
	Generator string `json:"generator"`
	Required  *bool  `json:"required"`
}

type herokuProc struct {
	Quantity int    `json:"quantity"`
	Size     string `json:"size"`
}

type herokuScripts struct {
	Postdeploy string `json:"postdeploy"`
}

type herokuAddon struct {
	Plan string
	As   string
}

func (a *herokuAddon) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil
	}
	if data[0] == '"' {
		return json.Unmarshal(data, &a.Plan)
	}
	var obj struct {
		Plan string `json:"plan"`
		As   string `json:"as"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	a.Plan = obj.Plan
	a.As = obj.As
	return nil
}

func convertHeroku(content []byte) (blueprint.DeploymentSpec, error) {
	var app herokuApp
	if err := json.Unmarshal(content, &app); err != nil {
		return blueprint.DeploymentSpec{}, fmt.Errorf("heroku: %w", err)
	}
	name := strings.TrimSpace(app.Name)
	if name == "" {
		return blueprint.DeploymentSpec{}, fmt.Errorf("heroku: name is required")
	}
	if len(app.Formation) == 0 {
		return blueprint.DeploymentSpec{}, fmt.Errorf("heroku: formation is required")
	}
	spec := newSpec(name, FormatHeroku)
	envs := herokuEnvs(app.Env)
	for _, procName := range sortedMapKeys(app.Formation) {
		spec.Components = append(spec.Components, herokuProcess(procName, app.Formation[procName], app.Image, envs, &spec))
	}
	addonNames := addHerokuAddons(&spec, app.Addons)
	addHerokuRelease(&spec, app.Scripts.Postdeploy)
	wireAddonDeps(&spec, addonNames)
	return spec, nil
}

func herokuProcess(name string, proc herokuProc, image string, envs []envIn, spec *blueprint.DeploymentSpec) blueprint.Component {
	replicas := proc.Quantity
	if replicas < 1 {
		replicas = 1
	}
	component := blueprint.Component{
		Name:  name,
		Kind:  blueprint.ComponentApp,
		Image: image,
		Process: blueprint.Process{
			Type: herokuProcessType(name),
		},
		Scaling: blueprint.Scaling{Replicas: replicas},
	}
	for _, env := range envs {
		addEnv(spec, &component, env)
	}
	return component
}

func herokuProcessType(name string) blueprint.ProcessType {
	if strings.EqualFold(name, "web") {
		return blueprint.ProcessTypeWeb
	}
	return blueprint.ProcessTypeWorker
}

func herokuEnvs(raw map[string]herokuEnv) []envIn {
	out := make([]envIn, 0, len(raw))
	for _, name := range sortedMapKeys(raw) {
		item := raw[name]
		required := true
		if item.Required != nil {
			required = *item.Required
		}
		value := item.Value
		if item.Generator != "" {
			value = ""
			required = true
		}
		out = append(out, envIn{Name: name, Value: value, Required: required})
	}
	return out
}

func addHerokuAddons(spec *blueprint.DeploymentSpec, addons []herokuAddon) []string {
	names := make([]string, 0, len(addons))
	for _, addon := range addons {
		component, ok := herokuAddonComponent(addon)
		if !ok {
			spec.Metadata = append(spec.Metadata, blueprint.Label{
				Name:  "unsupported_addon",
				Value: addon.Plan,
			})
			continue
		}
		spec.Components = append(spec.Components, component)
		names = append(names, component.Name)
	}
	return names
}

func herokuAddonComponent(addon herokuAddon) (blueprint.Component, bool) {
	plan := strings.ToLower(strings.TrimSpace(addon.Plan))
	base, _, _ := strings.Cut(plan, ":")
	image, path, fallback, ok := addonRuntime(base)
	if !ok {
		return blueprint.Component{}, false
	}
	name := addonName(addon.As, fallback)
	return blueprint.Component{
		Name:  name,
		Kind:  blueprint.ComponentStateful,
		Image: image,
		Stateful: &blueprint.Stateful{
			DataPath:   path,
			VolumeName: name + "-data",
		},
	}, true
}

func addonRuntime(plan string) (image, path, fallback string, ok bool) {
	switch plan {
	case "heroku-postgresql":
		return "postgres:16", "/var/lib/postgresql/data", "postgres", true
	case "heroku-redis":
		return "redis:7", "/data", "redis", true
	default:
		return "", "", "", false
	}
}

func addonName(as, fallback string) string {
	name := strings.ToLower(strings.TrimSpace(as))
	name = strings.ReplaceAll(name, "_", "-")
	if name == "" {
		return fallback
	}
	return name
}

func addHerokuRelease(spec *blueprint.DeploymentSpec, command string) {
	command = strings.TrimSpace(command)
	if command == "" {
		return
	}
	spec.Release.Commands = append(spec.Release.Commands, blueprint.ReleaseCommand{
		Name:    "postdeploy",
		Command: splitCommand(command),
	})
}

func wireAddonDeps(spec *blueprint.DeploymentSpec, addons []string) {
	if len(addons) == 0 {
		return
	}
	deps := make([]blueprint.DependsOn, 0, len(addons))
	for _, name := range addons {
		deps = append(deps, blueprint.DependsOn{
			Component: name,
			Condition: blueprint.DependencyConditionHealthy,
		})
	}
	for i := range spec.Components {
		if spec.Components[i].Kind != blueprint.ComponentApp {
			continue
		}
		spec.Components[i].Process.DependsOn = append(spec.Components[i].Process.DependsOn, deps...)
	}
	for i := range spec.Release.Commands {
		spec.Release.Commands[i].DependsOn = append(spec.Release.Commands[i].DependsOn, deps...)
	}
}
