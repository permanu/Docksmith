package swarm

import (
	"strconv"
	"strings"

	"github.com/permanu/docksmith/blueprint"
)

// StackFromBlueprint converts a resolved blueprint plan into the neutral Swarm
// stack consumed by RenderStack.
func StackFromBlueprint(plan blueprint.DeploymentPlan) (Stack, error) {
	if err := plan.Validate(); err != nil {
		return Stack{}, err
	}

	spec := plan.Spec
	stack := Stack{Version: "3.9"}
	secretSeen := make(map[string]struct{}, len(spec.Secrets))
	for _, s := range spec.Secrets {
		if s.Name == "" {
			continue
		}
		stack.Secrets = append(stack.Secrets, Secret{Name: s.Name, External: true})
		secretSeen[s.Name] = struct{}{}
	}

	netSeen := make(map[string]struct{})
	volumeSeen := make(map[string]struct{})
	addNetwork := func(name string) {
		if name == "" {
			return
		}
		if _, ok := netSeen[name]; ok {
			return
		}
		stack.Networks = append(stack.Networks, Network{Name: name, Driver: "overlay"})
		netSeen[name] = struct{}{}
	}
	for _, n := range spec.Placement.Networks {
		addNetwork(n)
	}

	components := plan.Components
	if len(components) == 0 {
		for _, c := range spec.Components {
			components = append(components, blueprint.ComponentPlan{
				Component: c,
				Mode:      spec.Mode,
				Placement: mergePlacement(spec.Placement, c.Placement),
			})
		}
	}

	for _, cp := range components {
		c := cp.Component
		placement := mergePlacement(spec.Placement, c.Placement)
		placement = mergePlacement(placement, cp.Placement)
		for _, n := range placement.Networks {
			addNetwork(n)
		}

		svc := Service{
			Name:                 c.Name,
			Image:                c.Image,
			Command:              appendCommand(c.Process.Command, c.Process.Args),
			Environment:          envMap(c.Env),
			Ports:                ports(c.Process.Ports),
			Secrets:              serviceSecrets(c.Secrets, c.Env),
			Networks:             uniqueStrings(placement.Networks),
			Deploy:               deploy(c, spec.Upgrade, placement),
			Healthcheck:          healthcheck(c.Image, c.Health, c.Process.Ports),
			Stateful:             c.Kind == blueprint.ComponentStateful,
			StatefulReplicasSafe: c.Capabilities.StatefulHA || (c.Stateful != nil && c.Stateful.HA),
		}
		for _, sec := range c.Secrets {
			if sec.Name == "" {
				continue
			}
			if _, ok := secretSeen[sec.Name]; ok {
				continue
			}
			stack.Secrets = append(stack.Secrets, Secret{Name: sec.Name, External: true})
			secretSeen[sec.Name] = struct{}{}
		}
		for _, env := range c.Env {
			secretName, ok := secretValueFromName(env.ValueFrom)
			if !ok {
				continue
			}
			if _, ok := secretSeen[secretName]; ok {
				continue
			}
			stack.Secrets = append(stack.Secrets, Secret{Name: secretName, External: true})
			secretSeen[secretName] = struct{}{}
		}
		if c.Stateful != nil {
			volume := c.Stateful.VolumeName
			if volume == "" {
				volume = c.Name + "-data"
			}
			svc.Mounts = append(svc.Mounts, Mount{
				Type:   "volume",
				Source: volume,
				Target: c.Stateful.DataPath,
			})
			if _, ok := volumeSeen[volume]; !ok {
				stack.Volumes = append(stack.Volumes, Volume{Name: volume})
				volumeSeen[volume] = struct{}{}
			}
		}
		stack.Services = append(stack.Services, svc)
	}

	return stack, nil
}

// RenderBlueprintStack converts and renders a blueprint deployment plan.
func RenderBlueprintStack(plan blueprint.DeploymentPlan) (string, error) {
	stack, err := StackFromBlueprint(plan)
	if err != nil {
		return "", err
	}
	return RenderStack(stack)
}

func appendCommand(command, args []string) []string {
	if len(command) == 0 {
		return append([]string(nil), args...)
	}
	out := append([]string(nil), command...)
	return append(out, args...)
}

func envMap(env []blueprint.Env) map[string]string {
	if len(env) == 0 {
		return nil
	}
	out := make(map[string]string, len(env))
	for _, e := range env {
		if e.ValueFrom == "" {
			out[e.Name] = e.Value
		}
	}
	return out
}

func ports(in []blueprint.Port) []Port {
	out := make([]Port, 0, len(in))
	for _, p := range in {
		out = append(out, Port{
			Target:    p.Container,
			Published: p.Published,
			Protocol:  p.Protocol,
		})
	}
	return out
}

func serviceSecrets(mounts []blueprint.SecretMount, env []blueprint.Env) []ServiceSecret {
	out := make([]ServiceSecret, 0, len(mounts)+len(env))
	seen := make(map[string]struct{}, len(mounts)+len(env))
	for _, s := range mounts {
		ref := ServiceSecret{Source: s.Name}
		if s.FilePath != "" {
			ref.Target = s.FilePath
		}
		seen[s.Name] = struct{}{}
		out = append(out, ref)
	}
	for _, e := range env {
		name, ok := secretValueFromName(e.ValueFrom)
		if !ok {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, ServiceSecret{Source: name})
	}
	return out
}

func deploy(c blueprint.Component, global blueprint.Upgrade, placement blueprint.Placement) Deploy {
	replicas := c.Scaling.Replicas
	if replicas == 0 {
		replicas = 1
	}
	upgrade := global
	if c.Upgrade.Strategy != "" || c.Upgrade.Order != "" || c.Upgrade.Parallelism > 0 || c.Upgrade.Delay != "" || c.Upgrade.FailureAction != "" || c.Upgrade.Monitor != "" || c.Upgrade.MaxFailureRatio > 0 {
		upgrade = c.Upgrade
	}
	return Deploy{
		Replicas: replicas,
		UpdateConfig: RolloutConfig{
			Parallelism:     upgrade.Parallelism,
			Delay:           upgrade.Delay,
			FailureAction:   upgrade.FailureAction,
			Monitor:         upgrade.Monitor,
			MaxFailureRatio: formatRatio(upgrade.MaxFailureRatio),
			Order:           upgrade.Order,
		},
		RollbackConfig: RolloutConfig{
			Parallelism:   upgrade.Parallelism,
			Delay:         upgrade.Delay,
			FailureAction: "pause",
			Monitor:       upgrade.Monitor,
			Order:         upgrade.Order,
		},
		Placement: Placement{Constraints: append([]string(nil), placement.Constraints...)},
	}
}

func healthcheck(image string, h blueprint.Health, ports []blueprint.Port) *Healthcheck {
	if len(h.Command) == 0 && h.Path == "" {
		return nil
	}
	hc := &Healthcheck{
		Test:        append([]string(nil), h.Command...),
		Interval:    h.Interval,
		Timeout:     h.Timeout,
		StartPeriod: h.StartPeriod,
		Retries:     h.Retries,
	}
	if len(hc.Test) == 0 {
		if !canUseGeneratedWgetHealthcheck(image) {
			return nil
		}
		port := 80
		if len(ports) > 0 && ports[0].Container > 0 {
			port = ports[0].Container
		}
		hc.Test = []string{"CMD-SHELL", "wget -qO- http://localhost:" + strconv.Itoa(port) + h.Path + " >/dev/null || exit 1"}
	}
	return hc
}

func secretValueFromName(valueFrom string) (string, bool) {
	name, ok := strings.CutPrefix(valueFrom, "secret:")
	return name, ok && name != ""
}

func canUseGeneratedWgetHealthcheck(image string) bool {
	normalized := strings.ToLower(image)
	return !strings.Contains(normalized, "distroless") && !strings.Contains(normalized, "scratch")
}

func mergePlacement(a, b blueprint.Placement) blueprint.Placement {
	return blueprint.Placement{
		Constraints: uniqueStrings(append(append([]string(nil), a.Constraints...), b.Constraints...)),
		Preferences: uniqueStrings(append(append([]string(nil), a.Preferences...), b.Preferences...)),
		Networks:    uniqueStrings(append(append([]string(nil), a.Networks...), b.Networks...)),
	}
}

func formatRatio(v float64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
