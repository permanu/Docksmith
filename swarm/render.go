package swarm

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const maxComposeNameLength = 63

var composeNameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,62}$`)

// RenderStack validates and renders spec as deterministic Docker Swarm stack
// YAML suitable for docker stack deploy.
func RenderStack(spec Stack) (string, error) {
	if err := Validate(spec); err != nil {
		return "", err
	}

	var b strings.Builder
	b.Grow(1024 + len(spec.Services)*512)

	version := spec.Version
	if version == "" {
		version = "3.9"
	}
	writeKV(&b, 0, "version", version)
	writeServices(&b, spec.Services)
	writeTopLevelSecrets(&b, spec.Secrets)
	writeTopLevelConfigs(&b, spec.Configs)
	writeTopLevelNetworks(&b, spec.Networks)
	writeTopLevelVolumes(&b, spec.Volumes)
	return b.String(), nil
}

func Validate(spec Stack) error {
	names := make(map[string]struct{}, len(spec.Services))
	for _, svc := range spec.Services {
		if err := validateComposeName("service", svc.Name); err != nil {
			return err
		}
		if svc.Image == "" {
			return fmt.Errorf("%w: service %q image is required", ErrInvalidSpec, svc.Name)
		}
		if _, ok := names[svc.Name]; ok {
			return fmt.Errorf("%w: duplicate service %q", ErrInvalidSpec, svc.Name)
		}
		names[svc.Name] = struct{}{}
		replicas := svc.Deploy.Replicas
		if replicas == 0 {
			replicas = 1
		}
		if svc.Stateful && replicas > 1 && !svc.StatefulReplicasSafe {
			return fmt.Errorf("%w: service %q requests %d replicas", ErrUnsupportedStatefulScale, svc.Name, replicas)
		}
		for _, secret := range svc.Secrets {
			if err := validateComposeName("service "+svc.Name+" secret source", secret.Source); err != nil {
				return err
			}
		}
		for _, config := range svc.Configs {
			if err := validateComposeName("service "+svc.Name+" config source", config.Source); err != nil {
				return err
			}
		}
		for _, network := range svc.Networks {
			if err := validateComposeName("service "+svc.Name+" network", network); err != nil {
				return err
			}
		}
		for _, port := range svc.Ports {
			if port.Target <= 0 {
				return fmt.Errorf("%w: service %q port target must be > 0", ErrInvalidSpec, svc.Name)
			}
		}
		for _, m := range svc.Mounts {
			if m.Type == "" || m.Target == "" {
				return fmt.Errorf("%w: service %q mount type and target are required", ErrInvalidSpec, svc.Name)
			}
			if m.Source != "" && m.Type == "volume" {
				if err := validateComposeName("service "+svc.Name+" volume source", m.Source); err != nil {
					return err
				}
			}
		}
	}
	secretNames, err := validateTopLevelSecrets(spec.Secrets)
	if err != nil {
		return err
	}
	configNames, err := validateTopLevelConfigs(spec.Configs)
	if err != nil {
		return err
	}
	networkNames, err := validateTopLevelNetworks(spec.Networks)
	if err != nil {
		return err
	}
	volumeNames, err := validateTopLevelVolumes(spec.Volumes)
	if err != nil {
		return err
	}
	for _, svc := range spec.Services {
		for _, secret := range svc.Secrets {
			if _, ok := secretNames[secret.Source]; !ok {
				return fmt.Errorf("%w: service %q references undeclared secret %q", ErrInvalidSpec, svc.Name, secret.Source)
			}
		}
		for _, config := range svc.Configs {
			if _, ok := configNames[config.Source]; !ok {
				return fmt.Errorf("%w: service %q references undeclared config %q", ErrInvalidSpec, svc.Name, config.Source)
			}
		}
		for _, network := range svc.Networks {
			if _, ok := networkNames[network]; !ok {
				return fmt.Errorf("%w: service %q references undeclared network %q", ErrInvalidSpec, svc.Name, network)
			}
		}
		for _, mount := range svc.Mounts {
			if mount.Type == "volume" && mount.Source != "" {
				if _, ok := volumeNames[mount.Source]; !ok {
					return fmt.Errorf("%w: service %q references undeclared volume %q", ErrInvalidSpec, svc.Name, mount.Source)
				}
			}
		}
	}
	return nil
}

func validateTopLevelSecrets(secrets []Secret) (map[string]struct{}, error) {
	names := make(map[string]struct{}, len(secrets))
	for _, secret := range secrets {
		if err := validateComposeName("secret", secret.Name); err != nil {
			return nil, err
		}
		if _, ok := names[secret.Name]; ok {
			return nil, fmt.Errorf("%w: duplicate secret %q", ErrInvalidSpec, secret.Name)
		}
		names[secret.Name] = struct{}{}
	}
	return names, nil
}

func validateTopLevelConfigs(configs []Config) (map[string]struct{}, error) {
	names := make(map[string]struct{}, len(configs))
	for _, config := range configs {
		if err := validateComposeName("config", config.Name); err != nil {
			return nil, err
		}
		if _, ok := names[config.Name]; ok {
			return nil, fmt.Errorf("%w: duplicate config %q", ErrInvalidSpec, config.Name)
		}
		names[config.Name] = struct{}{}
	}
	return names, nil
}

func validateTopLevelNetworks(networks []Network) (map[string]struct{}, error) {
	names := make(map[string]struct{}, len(networks))
	for _, network := range networks {
		if err := validateComposeName("network", network.Name); err != nil {
			return nil, err
		}
		if _, ok := names[network.Name]; ok {
			return nil, fmt.Errorf("%w: duplicate network %q", ErrInvalidSpec, network.Name)
		}
		names[network.Name] = struct{}{}
	}
	return names, nil
}

func validateTopLevelVolumes(volumes []Volume) (map[string]struct{}, error) {
	names := make(map[string]struct{}, len(volumes))
	for _, volume := range volumes {
		if err := validateComposeName("volume", volume.Name); err != nil {
			return nil, err
		}
		if _, ok := names[volume.Name]; ok {
			return nil, fmt.Errorf("%w: duplicate volume %q", ErrInvalidSpec, volume.Name)
		}
		names[volume.Name] = struct{}{}
	}
	return names, nil
}

func validateComposeName(kind, name string) error {
	if name == "" {
		return fmt.Errorf("%w: %s name is required", ErrInvalidSpec, kind)
	}
	if len(name) > maxComposeNameLength || !composeNameRE.MatchString(name) {
		return fmt.Errorf("%w: %s name %q must match %s and be at most %d characters", ErrInvalidSpec, kind, name, composeNameRE.String(), maxComposeNameLength)
	}
	return nil
}

func writeServices(b *strings.Builder, services []Service) {
	if len(services) == 0 {
		return
	}
	b.WriteString("services:\n")
	ordered := append([]Service(nil), services...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Name < ordered[j].Name })
	for _, svc := range ordered {
		writeKey(b, 1, svc.Name)
		writeKV(b, 2, "image", svc.Image)
		writeStringListKV(b, 2, "command", svc.Command)
		writeStringListKV(b, 2, "entrypoint", svc.Entrypoint)
		writeStringMap(b, 2, "environment", svc.Environment)
		writeStringMap(b, 2, "labels", svc.Labels)
		writePorts(b, svc.Ports)
		writeMounts(b, svc.Mounts)
		writeServiceSecrets(b, svc.Secrets)
		writeServiceConfigs(b, svc.Configs)
		writeStringList(b, 2, "networks", svc.Networks)
		writeDeploy(b, svc.Deploy)
		writeHealthcheck(b, svc.Healthcheck)
	}
}

func writePorts(b *strings.Builder, ports []Port) {
	if len(ports) == 0 {
		return
	}
	b.WriteString("    ports:\n")
	ordered := append([]Port(nil), ports...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Published == ordered[j].Published {
			return ordered[i].Target < ordered[j].Target
		}
		return ordered[i].Published < ordered[j].Published
	})
	for _, p := range ordered {
		b.WriteString("      - target: ")
		b.WriteString(strconv.Itoa(p.Target))
		b.WriteByte('\n')
		if p.Published > 0 {
			writeIntKV(b, 4, "published", p.Published)
		}
		if p.Protocol != "" {
			writeKV(b, 4, "protocol", p.Protocol)
		}
		if p.Mode != "" {
			writeKV(b, 4, "mode", p.Mode)
		}
	}
}

func writeMounts(b *strings.Builder, mounts []Mount) {
	if len(mounts) == 0 {
		return
	}
	b.WriteString("    volumes:\n")
	ordered := append([]Mount(nil), mounts...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Target == ordered[j].Target {
			return ordered[i].Source < ordered[j].Source
		}
		return ordered[i].Target < ordered[j].Target
	})
	for _, m := range ordered {
		b.WriteString("      - type: ")
		b.WriteString(quote(m.Type))
		b.WriteByte('\n')
		if m.Source != "" {
			writeKV(b, 4, "source", m.Source)
		}
		writeKV(b, 4, "target", m.Target)
		if m.ReadOnly {
			writeBoolKV(b, 4, "read_only", true)
		}
	}
}

func writeServiceSecrets(b *strings.Builder, secrets []ServiceSecret) {
	if len(secrets) == 0 {
		return
	}
	b.WriteString("    secrets:\n")
	ordered := append([]ServiceSecret(nil), secrets...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Source < ordered[j].Source })
	for _, s := range ordered {
		b.WriteString("      - source: ")
		b.WriteString(quote(s.Source))
		b.WriteByte('\n')
		writeOptionalKV(b, 4, "target", s.Target)
		writeOptionalKV(b, 4, "uid", s.UID)
		writeOptionalKV(b, 4, "gid", s.GID)
		writeOptionalKV(b, 4, "mode", s.Mode)
	}
}

func writeServiceConfigs(b *strings.Builder, configs []ServiceConfig) {
	if len(configs) == 0 {
		return
	}
	b.WriteString("    configs:\n")
	ordered := append([]ServiceConfig(nil), configs...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Source < ordered[j].Source })
	for _, c := range ordered {
		b.WriteString("      - source: ")
		b.WriteString(quote(c.Source))
		b.WriteByte('\n')
		writeOptionalKV(b, 4, "target", c.Target)
		writeOptionalKV(b, 4, "uid", c.UID)
		writeOptionalKV(b, 4, "gid", c.GID)
		writeOptionalKV(b, 4, "mode", c.Mode)
	}
}

func writeDeploy(b *strings.Builder, d Deploy) {
	if d.Replicas == 0 && isZeroResources(d.Resources) && isZeroRollout(d.UpdateConfig) && isZeroRollout(d.RollbackConfig) && len(d.Placement.Constraints) == 0 {
		return
	}
	b.WriteString("    deploy:\n")
	if d.Replicas > 0 {
		writeIntKV(b, 3, "replicas", d.Replicas)
	}
	writeResources(b, d.Resources)
	writeRollout(b, 3, "update_config", d.UpdateConfig)
	writeRollout(b, 3, "rollback_config", d.RollbackConfig)
	if len(d.Placement.Constraints) > 0 {
		b.WriteString("      placement:\n")
		writeStringList(b, 4, "constraints", d.Placement.Constraints)
	}
}

func writeResources(b *strings.Builder, r Resources) {
	if isZeroResources(r) {
		return
	}
	b.WriteString("      resources:\n")
	writeResourceSpec(b, 4, "limits", r.Limits)
	writeResourceSpec(b, 4, "reservations", r.Reservations)
}

func writeResourceSpec(b *strings.Builder, level int, key string, r ResourceSpec) {
	if r.CPUs == "" && r.Memory == "" {
		return
	}
	writeKey(b, level, key)
	writeOptionalKV(b, level+1, "cpus", r.CPUs)
	writeOptionalKV(b, level+1, "memory", r.Memory)
}

func writeRollout(b *strings.Builder, level int, key string, c RolloutConfig) {
	if isZeroRollout(c) {
		return
	}
	writeKey(b, level, key)
	if c.Parallelism > 0 {
		writeIntKV(b, level+1, "parallelism", c.Parallelism)
	}
	writeOptionalKV(b, level+1, "delay", c.Delay)
	writeOptionalKV(b, level+1, "failure_action", c.FailureAction)
	writeOptionalKV(b, level+1, "monitor", c.Monitor)
	writeOptionalKV(b, level+1, "max_failure_ratio", c.MaxFailureRatio)
	writeOptionalKV(b, level+1, "order", c.Order)
}

func writeHealthcheck(b *strings.Builder, hc *Healthcheck) {
	if hc == nil {
		return
	}
	b.WriteString("    healthcheck:\n")
	writeStringListKV(b, 3, "test", hc.Test)
	writeOptionalKV(b, 3, "interval", hc.Interval)
	writeOptionalKV(b, 3, "timeout", hc.Timeout)
	writeOptionalKV(b, 3, "start_period", hc.StartPeriod)
	if hc.Retries > 0 {
		writeIntKV(b, 3, "retries", hc.Retries)
	}
}

func writeTopLevelSecrets(b *strings.Builder, secrets []Secret) {
	if len(secrets) == 0 {
		return
	}
	b.WriteString("secrets:\n")
	ordered := append([]Secret(nil), secrets...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Name < ordered[j].Name })
	for _, s := range ordered {
		writeKey(b, 1, s.Name)
		writeExternalOrFile(b, s.External, s.File)
	}
}

func writeTopLevelConfigs(b *strings.Builder, configs []Config) {
	if len(configs) == 0 {
		return
	}
	b.WriteString("configs:\n")
	ordered := append([]Config(nil), configs...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Name < ordered[j].Name })
	for _, c := range ordered {
		writeKey(b, 1, c.Name)
		writeExternalOrFile(b, c.External, c.File)
	}
}

func writeTopLevelNetworks(b *strings.Builder, networks []Network) {
	if len(networks) == 0 {
		return
	}
	b.WriteString("networks:\n")
	ordered := append([]Network(nil), networks...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Name < ordered[j].Name })
	for _, n := range ordered {
		writeKey(b, 1, n.Name)
		if n.External {
			writeBoolKV(b, 2, "external", true)
		}
		if n.Driver != "" {
			writeKV(b, 2, "driver", n.Driver)
		}
		if !n.External && n.Driver == "" {
			b.WriteString("    driver: \"overlay\"\n")
		}
	}
}

func writeTopLevelVolumes(b *strings.Builder, volumes []Volume) {
	if len(volumes) == 0 {
		return
	}
	b.WriteString("volumes:\n")
	ordered := append([]Volume(nil), volumes...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Name < ordered[j].Name })
	for _, v := range ordered {
		writeKey(b, 1, v.Name)
		if v.External {
			writeBoolKV(b, 2, "external", true)
		}
		if v.Driver != "" {
			writeKV(b, 2, "driver", v.Driver)
		}
		if !v.External && v.Driver == "" {
			b.WriteString("    driver: \"local\"\n")
		}
	}
}

func writeExternalOrFile(b *strings.Builder, external bool, file string) {
	if external {
		writeBoolKV(b, 2, "external", true)
		return
	}
	if file != "" {
		writeKV(b, 2, "file", file)
	}
}

func writeStringMap(b *strings.Builder, level int, key string, values map[string]string) {
	if len(values) == 0 {
		return
	}
	writeKey(b, level, key)
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		writeKV(b, level+1, k, values[k])
	}
}

func writeStringList(b *strings.Builder, level int, key string, values []string) {
	if len(values) == 0 {
		return
	}
	writeKey(b, level, key)
	ordered := append([]string(nil), values...)
	sort.Strings(ordered)
	for _, v := range ordered {
		indent(b, level+1)
		b.WriteString("- ")
		b.WriteString(quote(v))
		b.WriteByte('\n')
	}
}

func writeStringListKV(b *strings.Builder, level int, key string, values []string) {
	if len(values) == 0 {
		return
	}
	indent(b, level)
	b.WriteString(key)
	b.WriteString(": [")
	for i, v := range values {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(quote(v))
	}
	b.WriteString("]\n")
}

func writeKey(b *strings.Builder, level int, key string) {
	indent(b, level)
	b.WriteString(key)
	b.WriteString(":\n")
}

func writeKV(b *strings.Builder, level int, key, value string) {
	indent(b, level)
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(quote(value))
	b.WriteByte('\n')
}

func writeOptionalKV(b *strings.Builder, level int, key, value string) {
	if value != "" {
		writeKV(b, level, key, value)
	}
}

func writeIntKV(b *strings.Builder, level int, key string, value int) {
	indent(b, level)
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(strconv.Itoa(value))
	b.WriteByte('\n')
}

func writeBoolKV(b *strings.Builder, level int, key string, value bool) {
	indent(b, level)
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(strconv.FormatBool(value))
	b.WriteByte('\n')
}

func indent(b *strings.Builder, level int) {
	for i := 0; i < level; i++ {
		b.WriteString("  ")
	}
}

func quote(s string) string {
	return strconv.Quote(s)
}

func isZeroResources(r Resources) bool {
	return r.Limits.CPUs == "" && r.Limits.Memory == "" && r.Reservations.CPUs == "" && r.Reservations.Memory == ""
}

func isZeroRollout(c RolloutConfig) bool {
	return c.Parallelism == 0 && c.Delay == "" && c.FailureAction == "" && c.Monitor == "" && c.MaxFailureRatio == "" && c.Order == ""
}
