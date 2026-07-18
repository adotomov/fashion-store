// Package telemetry wires the OpenTelemetry trace and metric pipelines that
// export to Google Cloud Trace and Cloud Monitoring. Both signals are opt-in
// (Config.TracesEnabled / MetricsEnabled): when both are off, Setup is a no-op
// so local and devbox runs need no GCP credentials or network egress.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"time"

	mexporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric"
	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	gcppropagator "github.com/GoogleCloudPlatform/opentelemetry-operations-go/propagator"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
)

// Config controls the telemetry pipeline. ProjectID is the GCP project spans
// and metrics are written to; it is required when either signal is enabled.
type Config struct {
	ProjectID      string
	ServiceName    string
	Env            string
	TracesEnabled  bool
	MetricsEnabled bool
	SampleRatio    float64
	MetricInterval time.Duration
}

// ShutdownFunc flushes and stops the telemetry providers. Always call it on
// process exit so buffered spans and the final metric batch are exported.
type ShutdownFunc func(context.Context) error

// Setup installs global OTel trace and/or metric providers per cfg and returns
// a shutdown func. When both signals are disabled it returns a no-op shutdown
// and installs nothing.
func Setup(ctx context.Context, cfg Config) (ShutdownFunc, error) {
	noop := func(context.Context) error { return nil }
	if !cfg.TracesEnabled && !cfg.MetricsEnabled {
		return noop, nil
	}
	if cfg.ProjectID == "" {
		return nil, errors.New("telemetry: GCP project id is required when tracing or metrics are enabled")
	}

	res := buildResource(ctx, cfg)

	var shutdownFns []func(context.Context) error

	if cfg.TracesEnabled {
		exp, err := texporter.New(texporter.WithProjectID(cfg.ProjectID))
		if err != nil {
			return nil, fmt.Errorf("telemetry: create trace exporter: %w", err)
		}
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exp),
			sdktrace.WithResource(res),
			// Parent-based: root spans are sampled at SampleRatio, children
			// follow the parent's decision so a trace is all-or-nothing.
			sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRatio))),
		)
		otel.SetTracerProvider(tp)
		shutdownFns = append(shutdownFns, tp.Shutdown)
	}

	if cfg.MetricsEnabled {
		exp, err := mexporter.New(mexporter.WithProjectID(cfg.ProjectID))
		if err != nil {
			return nil, fmt.Errorf("telemetry: create metric exporter: %w", err)
		}
		reader := sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(cfg.MetricInterval))
		mp := sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(reader),
			sdkmetric.WithResource(res),
		)
		otel.SetMeterProvider(mp)
		shutdownFns = append(shutdownFns, mp.Shutdown)
	}

	// Honour and emit Cloud Run's X-Cloud-Trace-Context (so our spans join the
	// load balancer / Cloud Run trace) alongside W3C tracecontext.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		gcppropagator.CloudTraceOneWayPropagator{},
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFns {
			if e := fn(ctx); e != nil {
				err = errors.Join(err, e)
			}
		}
		return err
	}, nil
}

// buildResource describes this process to Cloud Trace/Monitoring: the GCP
// detector adds Cloud Run service/revision labels; service name and environment
// are always set. A benign schema-URL conflict from the detector is ignored as
// long as a usable resource comes back.
func buildResource(ctx context.Context, cfg Config) *resource.Resource {
	res, err := resource.New(ctx,
		resource.WithDetectors(gcp.NewDetector()),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			attribute.String("deployment.environment", cfg.Env),
		),
	)
	if err != nil && res == nil {
		return resource.NewSchemaless(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			attribute.String("deployment.environment", cfg.Env),
		)
	}
	return res
}
