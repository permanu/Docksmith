package docksmith

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/permanu/docksmith/core"
	"github.com/permanu/docksmith/detect"
	"github.com/permanu/docksmith/plan"
)

// tracer returns the Docksmith OTel tracer. If no TracerProvider is installed
// (e.g. in unit tests), OTel's global default returns a no-op tracer — safe.
func tracer() trace.Tracer {
	return otel.Tracer("permanu/docksmith")
}

// detectWithSpan runs DetectWithOptions wrapped in a docksmith.classify span.
func detectWithSpan(ctx context.Context, dir string, opts detect.DetectOptions) (*core.Framework, error) {
	ctx, span := tracer().Start(ctx, "docksmith.classify")
	defer span.End()

	fw, err := detect.DetectWithOptions(dir, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetAttributes(
		attribute.String("docksmith.detected_language", fw.Name),
		attribute.Int64("docksmith.detected_port", int64(fw.Port)),
	)
	return fw, nil
}

// planWithSpan runs plan.Plan wrapped in a docksmith.plan span.
func planWithSpan(ctx context.Context, fw *core.Framework, opts ...plan.PlanOption) (*core.BuildPlan, error) {
	_, span := tracer().Start(ctx, "docksmith.plan")
	defer span.End()

	p, err := plan.Plan(fw, opts...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetAttributes(
		attribute.String("docksmith.detected_language", fw.Name),
		attribute.Int64("docksmith.layers_count", int64(len(p.Stages))),
	)
	return p, nil
}

// resolveBaseImageDigestWithSpan calls ResolveBaseImageDigest wrapped in a
// docksmith.cache_check span (digest resolution is effectively a registry
// cache probe).
func resolveBaseImageDigestWithSpan(ctx context.Context, baseRef string) (string, error) {
	ctx, span := tracer().Start(ctx, "docksmith.cache_check")
	defer span.End()

	span.SetAttributes(attribute.String("docksmith.image_tag", baseRef))

	digest, err := plan.ResolveBaseImageDigest(ctx, baseRef)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}
	span.SetAttributes(attribute.String("docksmith.digest", digest))
	return digest, nil
}
