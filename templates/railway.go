// SPDX-License-Identifier: MPL-2.0

package templates

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/permanu/docksmith/blueprint"
)

type railwayDoc struct {
	Name     string          `json:"name"`
	Services json.RawMessage `json:"services"`
}

type railwayService struct {
	Name          string                     `json:"name"`
	Image         string                     `json:"image"`
	RootDirectory string                     `json:"rootDirectory"`
	Source        *railwaySource             `json:"source"`
	Build         *railwayBuild              `json:"build"`
	Deploy        *railwayDeploy             `json:"deploy"`
	Variables     map[string]json.RawMessage `json:"variables"`
	VolumeMounts  json.RawMessage            `json:"volumeMounts"`
	Domains       []railwayDomain            `json:"domains"`
}

type railwaySource struct {
	Image         string `json:"image"`
	Repo          string `json:"repo"`
	RootDirectory string `json:"rootDirectory"`
}

type railwayBuild struct {
	BuildCommand string `json:"buildCommand"`
}

type railwayDeploy struct {
	StartCommand    string `json:"startCommand"`
	HealthcheckPath string `json:"healthcheckPath"`
	NumReplicas     int    `json:"numReplicas"`
}

type railwayDomain struct {
	TargetPort int `json:"targetPort"`
}

type railwayVarObj struct {
	DefaultValue *string `json:"defaultValue"`
	Default      *string `json:"default"`
	Value        *string `json:"value"`
	IsOptional   *bool   `json:"isOptional"`
	Required     *bool   `json:"required"`
	Generator    string  `json:"generator"`
}

type railwayPending struct {
	index int
	envs  []envIn
}

func convertRailway(content []byte) (blueprint.DeploymentSpec, error) {
	var doc railwayDoc
	if err := json.Unmarshal(content, &doc); err != nil {
		return blueprint.DeploymentSpec{}, fmt.Errorf("railway: %w", err)
	}
	services, err := decodeRailwayServices(doc.Services)
	if err != nil {
		return blueprint.DeploymentSpec{}, err
	}
	name := strings.TrimSpace(doc.Name)
	if name == "" {
		if len(services) != 1 {
			return blueprint.DeploymentSpec{}, errors.New("railway: name is required")
		}
		name = services[0].Name
	}
	spec := newSpec(name, FormatRailway)
	pending := make([]railwayPending, 0, len(services))
	for i := range services {
		component, envs, err := railwayComponent(services[i])
		if err != nil {
			return blueprint.DeploymentSpec{}, err
		}
		pending = append(pending, railwayPending{index: len(spec.Components), envs: envs})
		spec.Components = append(spec.Components, component)
	}
	for _, item := range pending {
		for _, env := range item.envs {
			addEnv(&spec, &spec.Components[item.index], env)
		}
		wireServiceRefs(&spec, item.index, item.envs)
	}
	return spec, nil
}

func railwayComponent(svc railwayService) (blueprint.Component, []envIn, error) {
	name := strings.TrimSpace(svc.Name)
	if name == "" {
		return blueprint.Component{}, nil, errors.New("railway: service name is required")
	}
	image, root := railwaySourceOf(svc)
	volumes, err := decodeRailwayVolumes(svc.VolumeMounts)
	if err != nil {
		return blueprint.Component{}, nil, err
	}
	envs, err := decodeRailwayEnvs(svc.Variables)
	if err != nil {
		return blueprint.Component{}, nil, err
	}
	component := blueprint.Component{
		Name:  name,
		Image: image,
		Kind:  blueprint.ComponentApp,
		Process: blueprint.Process{
			WorkingDir: repoPath(root),
		},
	}
	if isDatabaseImage(image) {
		applyStateful(&component, volumes)
	}
	applyRailwayDeploy(&component, svc.Deploy)
	if port := railwayPort(svc, envs); port > 0 && component.Kind == blueprint.ComponentApp {
		component.Process.Type = blueprint.ProcessTypeWeb
		component.Process.Ports = []blueprint.Port{{Name: "http", Container: port, Protocol: "tcp"}}
	}
	return component, envs, nil
}

func railwaySourceOf(svc railwayService) (image, root string) {
	image = svc.Image
	root = svc.RootDirectory
	if svc.Source == nil {
		return image, root
	}
	if image == "" {
		image = svc.Source.Image
	}
	if root == "" {
		root = svc.Source.RootDirectory
	}
	return image, root
}

func applyRailwayDeploy(component *blueprint.Component, deploy *railwayDeploy) {
	if deploy == nil {
		return
	}
	if deploy.StartCommand != "" {
		component.Process.Command = splitCommand(deploy.StartCommand)
	}
	if deploy.HealthcheckPath != "" {
		component.Health.Path = deploy.HealthcheckPath
	}
	if deploy.NumReplicas > 0 {
		component.Scaling.Replicas = deploy.NumReplicas
	}
}

func railwayPort(svc railwayService, envs []envIn) int {
	for _, domain := range svc.Domains {
		if domain.TargetPort > 0 {
			return domain.TargetPort
		}
	}
	for _, env := range envs {
		if env.Name != "PORT" || env.Value == "" {
			continue
		}
		var port int
		if _, err := fmt.Sscan(env.Value, &port); err == nil && port > 0 && port < 65536 {
			return port
		}
	}
	return 0
}

func decodeRailwayServices(raw json.RawMessage) ([]railwayService, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, errors.New("railway: services are required")
	}
	if raw[0] == '[' {
		var list []railwayService
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, fmt.Errorf("railway: services: %w", err)
		}
		return list, nil
	}
	var keyed map[string]railwayService
	if err := json.Unmarshal(raw, &keyed); err != nil {
		return nil, fmt.Errorf("railway: services: %w", err)
	}
	keys := make([]string, 0, len(keyed))
	for key := range keyed {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]railwayService, 0, len(keys))
	for _, key := range keys {
		svc := keyed[key]
		if strings.TrimSpace(svc.Name) == "" {
			svc.Name = key
		}
		out = append(out, svc)
	}
	if len(out) == 0 {
		return nil, errors.New("railway: services are required")
	}
	return out, nil
}

func decodeRailwayVolumes(raw json.RawMessage) ([]namedVolume, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, nil
	}
	if raw[0] == '[' {
		return decodeRailwayVolumeList(raw)
	}
	var keyed map[string]struct {
		MountPath string `json:"mountPath"`
	}
	if err := json.Unmarshal(raw, &keyed); err != nil {
		return nil, fmt.Errorf("railway: volumeMounts: %w", err)
	}
	keys := make([]string, 0, len(keyed))
	for key := range keyed {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]namedVolume, 0, len(keys))
	for _, key := range keys {
		out = append(out, namedVolume{Name: key, MountPath: keyed[key].MountPath})
	}
	return out, nil
}

func decodeRailwayVolumeList(raw json.RawMessage) ([]namedVolume, error) {
	var list []struct {
		Name      string `json:"name"`
		MountPath string `json:"mountPath"`
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("railway: volumeMounts: %w", err)
	}
	out := make([]namedVolume, 0, len(list))
	for _, item := range list {
		out = append(out, namedVolume{Name: item.Name, MountPath: item.MountPath})
	}
	return out, nil
}

func decodeRailwayEnvs(raw map[string]json.RawMessage) ([]envIn, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]envIn, 0, len(keys))
	for _, key := range keys {
		env, err := decodeRailwayVar(key, raw[key])
		if err != nil {
			return nil, err
		}
		out = append(out, env)
	}
	return out, nil
}

func decodeRailwayVar(name string, raw json.RawMessage) (envIn, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return envIn{Name: name, Required: true}, nil
	}
	switch raw[0] {
	case '"':
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return envIn{}, fmt.Errorf("railway: variable %s: %w", name, err)
		}
		return envIn{Name: name, Value: value, Required: value == ""}, nil
	case '{':
		var obj railwayVarObj
		if err := json.Unmarshal(raw, &obj); err != nil {
			return envIn{}, fmt.Errorf("railway: variable %s: %w", name, err)
		}
		return railwayVarEnv(name, obj), nil
	default:
		return envIn{Name: name, Value: string(raw)}, nil
	}
}

func railwayVarEnv(name string, obj railwayVarObj) envIn {
	if obj.Generator != "" {
		required := true
		if obj.IsOptional != nil {
			required = !*obj.IsOptional
		}
		return envIn{Name: name, Required: required}
	}
	value, ok := firstString(obj.Value, obj.DefaultValue, obj.Default)
	required := !ok || value == ""
	switch {
	case obj.IsOptional != nil:
		required = !*obj.IsOptional
	case obj.Required != nil:
		required = *obj.Required
	}
	if !ok {
		value = ""
	}
	return envIn{Name: name, Value: value, Required: required}
}

func firstString(values ...*string) (string, bool) {
	for _, value := range values {
		if value != nil {
			return *value, true
		}
	}
	return "", false
}
