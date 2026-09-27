// SPDX-License-Identifier: MPL-2.0

package templates

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/permanu/docksmith/blueprint"
)

const (
	// FormatRailway is a Railway template.json document.
	FormatRailway = "railway"
	// FormatCompose is a Docker Compose file.
	FormatCompose = "compose"
	// FormatHeroku is a Heroku app.json document.
	FormatHeroku = "heroku"
	// FormatHelm is a Helm chart. Helm is experimental and is not imported.
	FormatHelm = "helm"

	schemaVersion    = "blueprint.docksmith.dev/v1"
	secretPrefix     = "secret:"
	maxTemplateBytes = 1 << 20
)

// ErrUnsupported is returned when a template format cannot be imported.
var ErrUnsupported = errors.New("unsupported template import")

// ErrExperimental marks a format that is not a supported import.
var ErrExperimental = errors.New("experimental template import")

// Convert turns one Railway template.json, Docker Compose file, or Heroku
// app.json into a blueprint deployment spec. Helm charts are marked
// experimental and are not a supported import.
func Convert(kind string, content []byte) (blueprint.DeploymentSpec, error) {
	format := normalizeKind(kind)
	if format == FormatHelm {
		return blueprint.DeploymentSpec{}, helmUnsupported()
	}
	if format != FormatRailway && format != FormatCompose && format != FormatHeroku {
		return blueprint.DeploymentSpec{}, fmt.Errorf("template format %q: %w", kind, ErrUnsupported)
	}
	if err := validateContent(content); err != nil {
		return blueprint.DeploymentSpec{}, err
	}
	var (
		spec blueprint.DeploymentSpec
		err  error
	)
	switch format {
	case FormatRailway:
		spec, err = convertRailway(content)
	case FormatCompose:
		spec, err = convertCompose(content)
	default:
		spec, err = convertHeroku(content)
	}
	if err != nil {
		return blueprint.DeploymentSpec{}, err
	}
	if err := spec.Validate(); err != nil {
		return blueprint.DeploymentSpec{}, fmt.Errorf("%s: %w", format, err)
	}
	return spec, nil
}

func normalizeKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case FormatRailway, "railway_template", "template.json", "template":
		return FormatRailway
	case FormatCompose, "docker-compose", "docker_compose", "compose.yaml", "compose.yml":
		return FormatCompose
	case FormatHeroku, "app_json", "app.json", "heroku_app":
		return FormatHeroku
	case FormatHelm, "chart", "chart.yaml", "helm_chart":
		return FormatHelm
	default:
		return ""
	}
}

func validateContent(content []byte) error {
	if len(bytes.TrimSpace(content)) == 0 {
		return errors.New("template: content is empty")
	}
	if len(content) > maxTemplateBytes {
		return fmt.Errorf("template: content exceeds %d bytes", maxTemplateBytes)
	}
	return nil
}

func helmUnsupported() error {
	return fmt.Errorf("helm charts are marked experimental and are not a supported import: %w", errors.Join(ErrExperimental, ErrUnsupported))
}

func newSpec(name, format string) blueprint.DeploymentSpec {
	return blueprint.DeploymentSpec{
		SchemaVersion: schemaVersion,
		Name:          name,
		Environment:   blueprint.EnvironmentDevelopment,
		Metadata: []blueprint.Label{{
			Name:  "import_format",
			Value: format,
		}},
	}
}
