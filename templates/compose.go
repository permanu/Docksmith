// SPDX-License-Identifier: MPL-2.0

package templates

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/permanu/docksmith/blueprint"
	"gopkg.in/yaml.v3"
)

type composeFile struct {
	Name     string                    `yaml:"name"`
	Services map[string]composeService `yaml:"services"`
}

type composeService struct {
	Image       string         `yaml:"image"`
	WorkingDir  string         `yaml:"working_dir"`
	Command     any            `yaml:"command"`
	Entrypoint  any            `yaml:"entrypoint"`
	Ports       []any          `yaml:"ports"`
	Environment any            `yaml:"environment"`
	DependsOn   any            `yaml:"depends_on"`
	Healthcheck *composeHealth `yaml:"healthcheck"`
	Deploy      *composeDeploy `yaml:"deploy"`
	Volumes     []any          `yaml:"volumes"`
}

type composeHealth struct {
	Test        any    `yaml:"test"`
	Interval    string `yaml:"interval"`
	Timeout     string `yaml:"timeout"`
	StartPeriod string `yaml:"start_period"`
	Retries     int    `yaml:"retries"`
}

type composeDeploy struct {
	Replicas int `yaml:"replicas"`
}

func convertCompose(content []byte) (blueprint.DeploymentSpec, error) {
	var file composeFile
	if err := yaml.Unmarshal(content, &file); err != nil {
		return blueprint.DeploymentSpec{}, fmt.Errorf("compose: %w", err)
	}
	name := strings.TrimSpace(file.Name)
	if name == "" {
		return blueprint.DeploymentSpec{}, fmt.Errorf("compose: name is required")
	}
	if len(file.Services) == 0 {
		return blueprint.DeploymentSpec{}, fmt.Errorf("compose: services are required")
	}
	spec := newSpec(name, FormatCompose)
	names := sortedMapKeys(file.Services)
	for _, serviceName := range names {
		component, err := composeComponent(serviceName, file.Services[serviceName])
		if err != nil {
			return blueprint.DeploymentSpec{}, err
		}
		spec.Components = append(spec.Components, component)
	}
	for i, serviceName := range names {
		if err := applyComposeEdges(&spec, i, file.Services[serviceName]); err != nil {
			return blueprint.DeploymentSpec{}, err
		}
	}
	return spec, nil
}

func composeComponent(name string, svc composeService) (blueprint.Component, error) {
	command, args, err := composeCommand(svc)
	if err != nil {
		return blueprint.Component{}, err
	}
	ports, err := composePorts(svc.Ports)
	if err != nil {
		return blueprint.Component{}, err
	}
	volumes, err := composeVolumes(svc.Volumes)
	if err != nil {
		return blueprint.Component{}, err
	}
	health, err := healthFromCompose(svc.Healthcheck)
	if err != nil {
		return blueprint.Component{}, err
	}
	component := blueprint.Component{
		Name:  name,
		Image: svc.Image,
		Kind:  blueprint.ComponentApp,
		Process: blueprint.Process{
			WorkingDir: svc.WorkingDir,
			Command:    command,
			Args:       args,
			Ports:      ports,
		},
		Health: health,
	}
	if svc.Deploy != nil && svc.Deploy.Replicas > 0 {
		component.Scaling.Replicas = svc.Deploy.Replicas
	}
	if isDatabaseImage(svc.Image) {
		applyStateful(&component, volumes)
	}
	if component.Kind == blueprint.ComponentApp && len(ports) > 0 {
		component.Process.Type = blueprint.ProcessTypeWeb
	}
	return component, nil
}

func composeCommand(svc composeService) (command, args []string, err error) {
	command, err = asStrings(svc.Entrypoint)
	if err != nil {
		return nil, nil, fmt.Errorf("compose: entrypoint: %w", err)
	}
	args, err = asStrings(svc.Command)
	if err != nil {
		return nil, nil, fmt.Errorf("compose: command: %w", err)
	}
	if len(command) == 0 {
		return args, nil, nil
	}
	return command, args, nil
}

func healthFromCompose(health *composeHealth) (blueprint.Health, error) {
	if health == nil {
		return blueprint.Health{}, nil
	}
	command, err := asStrings(health.Test)
	if err != nil {
		return blueprint.Health{}, fmt.Errorf("compose: healthcheck.test: %w", err)
	}
	return blueprint.Health{
		Command:     command,
		Interval:    health.Interval,
		Timeout:     health.Timeout,
		StartPeriod: health.StartPeriod,
		Retries:     health.Retries,
	}, nil
}

func applyComposeEdges(spec *blueprint.DeploymentSpec, index int, svc composeService) error {
	envs, err := parseEnv(svc.Environment)
	if err != nil {
		return err
	}
	for _, env := range envs {
		addEnv(spec, &spec.Components[index], env)
	}
	deps, err := parseDepends(svc.DependsOn)
	if err != nil {
		return err
	}
	for _, dep := range deps {
		addDepends(&spec.Components[index], dep.Component, dep.Condition)
	}
	return nil
}

func composePorts(raw []any) ([]blueprint.Port, error) {
	out := make([]blueprint.Port, 0, len(raw))
	for _, item := range raw {
		port, err := oneComposePort(item)
		if err != nil {
			return nil, err
		}
		if port.Container > 0 {
			out = append(out, port)
		}
	}
	return out, nil
}

func oneComposePort(raw any) (blueprint.Port, error) {
	switch value := raw.(type) {
	case string:
		return parsePortSpec(value)
	case map[string]any:
		return portFromMap(value)
	default:
		return blueprint.Port{}, fmt.Errorf("compose: port has unsupported type %T", raw)
	}
}

func parsePortSpec(spec string) (blueprint.Port, error) {
	spec = strings.TrimSpace(spec)
	protocol := ""
	if host, proto, ok := strings.Cut(spec, "/"); ok {
		spec = host
		protocol = proto
	}
	parts := strings.Split(spec, ":")
	var published, container string
	switch len(parts) {
	case 1:
		container = parts[0]
	case 2:
		published, container = parts[0], parts[1]
	case 3:
		published, container = parts[1], parts[2]
	default:
		return blueprint.Port{}, fmt.Errorf("compose: invalid port %q", spec)
	}
	containerPort, err := parsePortNumber(container)
	if err != nil {
		return blueprint.Port{}, fmt.Errorf("compose: container port: %w", err)
	}
	port := blueprint.Port{Name: "http", Container: containerPort, Protocol: protocol}
	if published == "" {
		return port, nil
	}
	publishedPort, err := parsePortNumber(published)
	if err != nil {
		return blueprint.Port{}, fmt.Errorf("compose: published port: %w", err)
	}
	port.Published = publishedPort
	return port, nil
}

func parsePortNumber(value string) (int, error) {
	port, err := strconv.Atoi(value)
	if err != nil || port <= 0 || port > 65535 {
		return 0, fmt.Errorf("invalid port %q", value)
	}
	return port, nil
}

func portFromMap(raw map[string]any) (blueprint.Port, error) {
	container, err := parsePortNumber(stringify(raw["target"]))
	if err != nil {
		return blueprint.Port{}, fmt.Errorf("compose: port target: %w", err)
	}
	port := blueprint.Port{Name: "http", Container: container, Protocol: stringify(raw["protocol"])}
	if published := stringify(raw["published"]); published != "" {
		n, err := parsePortNumber(published)
		if err != nil {
			return blueprint.Port{}, fmt.Errorf("compose: port published: %w", err)
		}
		port.Published = n
	}
	return port, nil
}

func composeVolumes(raw []any) ([]namedVolume, error) {
	out := make([]namedVolume, 0, len(raw))
	for _, item := range raw {
		switch value := item.(type) {
		case string:
			volume, err := parseVolumeSpec(value)
			if err != nil {
				return nil, err
			}
			out = append(out, volume)
		case map[string]any:
			out = append(out, namedVolume{Name: stringify(value["source"]), MountPath: stringify(value["target"])})
		default:
			return nil, fmt.Errorf("compose: volume has unsupported type %T", item)
		}
	}
	return out, nil
}

func parseVolumeSpec(spec string) (namedVolume, error) {
	parts := strings.Split(spec, ":")
	switch len(parts) {
	case 1:
		return namedVolume{MountPath: parts[0]}, nil
	case 2, 3:
		return namedVolume{Name: parts[0], MountPath: parts[1]}, nil
	default:
		return namedVolume{}, fmt.Errorf("compose: invalid volume %q", spec)
	}
}

func parseEnv(raw any) ([]envIn, error) {
	switch value := raw.(type) {
	case nil:
		return nil, nil
	case map[string]any:
		return envFromMap(value), nil
	case map[string]string:
		return envFromStrings(value), nil
	case []any:
		return envFromList(value)
	default:
		return nil, fmt.Errorf("compose: environment has unsupported type %T", raw)
	}
}

func envFromMap(values map[string]any) []envIn {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]envIn, 0, len(keys))
	for _, key := range keys {
		val := stringify(values[key])
		out = append(out, envIn{Name: key, Value: val, Required: val == ""})
	}
	return out
}

func envFromStrings(values map[string]string) []envIn {
	generic := make(map[string]any, len(values))
	for key, value := range values {
		generic[key] = value
	}
	return envFromMap(generic)
}

func envFromList(values []any) ([]envIn, error) {
	out := make([]envIn, 0, len(values))
	for _, item := range values {
		text, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("compose: environment entry has type %T", item)
		}
		key, value, _ := strings.Cut(text, "=")
		if key == "" {
			return nil, fmt.Errorf("compose: environment entry %q is missing a name", text)
		}
		out = append(out, envIn{Name: key, Value: value, Required: value == ""})
	}
	return out, nil
}

func parseDepends(raw any) ([]blueprint.DependsOn, error) {
	switch value := raw.(type) {
	case nil:
		return nil, nil
	case []any:
		return dependsFromList(value)
	case map[string]any:
		return dependsFromMap(value)
	default:
		return nil, fmt.Errorf("compose: depends_on has unsupported type %T", raw)
	}
}

func dependsFromList(values []any) ([]blueprint.DependsOn, error) {
	out := make([]blueprint.DependsOn, 0, len(values))
	for _, item := range values {
		name, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("compose: depends_on entry has type %T", item)
		}
		out = append(out, blueprint.DependsOn{Component: name, Condition: blueprint.DependencyConditionStarted})
	}
	return out, nil
}

func dependsFromMap(values map[string]any) ([]blueprint.DependsOn, error) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]blueprint.DependsOn, 0, len(keys))
	for _, key := range keys {
		condition, err := dependCondition(values[key])
		if err != nil {
			return nil, err
		}
		out = append(out, blueprint.DependsOn{Component: key, Condition: condition})
	}
	return out, nil
}

func dependCondition(raw any) (blueprint.DependencyCondition, error) {
	switch value := raw.(type) {
	case nil:
		return blueprint.DependencyConditionStarted, nil
	case string:
		return mapCondition(value)
	case map[string]any:
		return mapCondition(stringify(value["condition"]))
	default:
		return "", fmt.Errorf("compose: depends_on condition has type %T", raw)
	}
}

func mapCondition(value string) (blueprint.DependencyCondition, error) {
	switch value {
	case "", "service_started", "service_completed_successfully":
		return blueprint.DependencyConditionStarted, nil
	case "service_healthy":
		return blueprint.DependencyConditionHealthy, nil
	default:
		return "", fmt.Errorf("compose: unsupported depends_on condition %q", value)
	}
}

func asStrings(raw any) ([]string, error) {
	switch value := raw.(type) {
	case nil:
		return nil, nil
	case string:
		if strings.TrimSpace(value) == "" {
			return nil, nil
		}
		return strings.Fields(value), nil
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			text, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("expected string, got %T", item)
			}
			out = append(out, text)
		}
		return out, nil
	case []string:
		return append([]string(nil), value...), nil
	default:
		return nil, fmt.Errorf("expected string or list, got %T", raw)
	}
}

func stringify(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return fmt.Sprint(typed)
	}
}

func sortedMapKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
