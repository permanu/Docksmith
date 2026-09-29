// SPDX-License-Identifier: MPL-2.0

package templates

import (
	"regexp"
	"strings"

	"github.com/permanu/docksmith/blueprint"
)

type envIn struct {
	Name     string
	Value    string
	Required bool
}

type namedVolume struct {
	Name      string
	MountPath string
}

var sensitiveFragments = []string{
	"SECRET",
	"TOKEN",
	"PASSWORD",
	"PASSWD",
	"PRIVATE_KEY",
	"API_KEY",
	"ACCESS_KEY",
	"DATABASE",
	"CREDENTIAL",
}

var serviceRefPattern = regexp.MustCompile(`\$\{\{\s*([A-Za-z0-9][A-Za-z0-9_-]*)\s*\.`)

func addEnv(spec *blueprint.DeploymentSpec, component *blueprint.Component, env envIn) {
	name := strings.TrimSpace(env.Name)
	if name == "" {
		return
	}
	if env.Value == "" && !env.Required && !sensitiveName(name) {
		return
	}
	if env.Value != "" && !sensitiveName(name) && !secretTemplate(env.Value) {
		component.Env = append(component.Env, blueprint.Env{
			Name:     name,
			Value:    env.Value,
			Required: env.Required,
		})
		return
	}
	secret := secretName(name)
	addSecret(spec, secret, env.Required || sensitiveName(name))
	component.Env = append(component.Env, blueprint.Env{
		Name:      name,
		ValueFrom: secretPrefix + secret,
		Required:  env.Required || sensitiveName(name) || env.Value == "",
	})
}

func addSecret(spec *blueprint.DeploymentSpec, name string, required bool) {
	for i := range spec.Secrets {
		if spec.Secrets[i].Name == name {
			if required {
				spec.Secrets[i].Required = true
			}
			return
		}
	}
	spec.Secrets = append(spec.Secrets, blueprint.Secret{Name: name, Required: required})
}

func sensitiveName(name string) bool {
	upper := strings.ToUpper(name)
	for _, fragment := range sensitiveFragments {
		if strings.Contains(upper, fragment) {
			return true
		}
	}
	return upper == "KEY" || strings.HasSuffix(upper, "_KEY")
}

func secretTemplate(value string) bool {
	return strings.Contains(value, "${{") || strings.Contains(strings.ToLower(value), "secret(")
}

func secretName(envName string) string {
	return strings.ReplaceAll(strings.ToLower(envName), "_", "-")
}

func referencedServices(value string) []string {
	matches := serviceRefPattern.FindAllStringSubmatch(value, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		name := match[1]
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func addDepends(component *blueprint.Component, name string, condition blueprint.DependencyCondition) {
	if name == "" || name == component.Name {
		return
	}
	for i := range component.Process.DependsOn {
		if component.Process.DependsOn[i].Component == name {
			return
		}
	}
	component.Process.DependsOn = append(component.Process.DependsOn, blueprint.DependsOn{
		Component: name,
		Condition: condition,
	})
}

func conditionFor(components []blueprint.Component, name string) blueprint.DependencyCondition {
	for i := range components {
		if components[i].Name == name && components[i].Kind == blueprint.ComponentStateful {
			return blueprint.DependencyConditionHealthy
		}
	}
	return blueprint.DependencyConditionStarted
}

func wireServiceRefs(spec *blueprint.DeploymentSpec, index int, envs []envIn) {
	component := &spec.Components[index]
	known := make(map[string]struct{}, len(spec.Components))
	for i := range spec.Components {
		known[spec.Components[i].Name] = struct{}{}
	}
	for _, env := range envs {
		for _, ref := range referencedServices(env.Value) {
			if _, ok := known[ref]; !ok {
				continue
			}
			addDepends(component, ref, conditionFor(spec.Components, ref))
		}
	}
}

func applyStateful(component *blueprint.Component, volumes []namedVolume) {
	component.Kind = blueprint.ComponentStateful
	path := defaultDataPath(component.Image)
	volume := component.Name + "-data"
	if len(volumes) > 0 {
		if volumes[0].MountPath != "" {
			path = volumes[0].MountPath
		}
		if volumes[0].Name != "" {
			volume = volumes[0].Name
		}
	}
	component.Stateful = &blueprint.Stateful{DataPath: path, VolumeName: volume}
}

func isDatabaseImage(image string) bool {
	switch imageName(image) {
	case "postgres", "postgresql", "mysql", "mariadb", "redis", "mongo", "mongodb":
		return true
	default:
		return false
	}
}

func imageName(image string) string {
	image = strings.ToLower(strings.TrimSpace(image))
	if slash := strings.LastIndex(image, "/"); slash >= 0 {
		image = image[slash+1:]
	}
	if at := strings.Index(image, "@"); at >= 0 {
		image = image[:at]
	}
	if colon := strings.LastIndex(image, ":"); colon >= 0 {
		image = image[:colon]
	}
	return image
}

func defaultDataPath(image string) string {
	switch imageName(image) {
	case "postgres", "postgresql":
		return "/var/lib/postgresql/data"
	case "mysql", "mariadb":
		return "/var/lib/mysql"
	case "mongo", "mongodb":
		return "/data/db"
	default:
		return "/data"
	}
}

func splitCommand(command string) []string {
	return strings.Fields(strings.TrimSpace(command))
}

func repoPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "/")
	if path == "." {
		return ""
	}
	return path
}
