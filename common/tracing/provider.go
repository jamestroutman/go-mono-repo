// Spec: docs/specs/004-opentelemetry-tracing.md

package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	"github.com/getsentry/sentry-go"
	sentryotel "github.com/getsentry/sentry-go/otel"
)

// TracingConfig holds all configuration for distributed tracing
// Spec: docs/specs/004-opentelemetry-tracing.md#configuration-integration
type TracingConfig struct {
	Enabled        bool    `envconfig:"TRACING_ENABLED" default:"true"`
	SentryDSN      string  `envconfig:"SENTRY_DSN" default:""`
	SampleRate     float64 `envconfig:"TRACE_SAMPLE_RATE" default:"0.01"`  // 1% default for production safety
	Environment    string  `envconfig:"TRACE_ENVIRONMENT" default:""`       // Defaults to main Environment field
	ServiceName    string  `envconfig:"TRACE_SERVICE_NAME" default:""`      // Defaults to main ServiceName field
	ServiceVersion string  `envconfig:"TRACE_SERVICE_VERSION" default:""`   // Defaults to main ServiceVersion field
}

// GetEnvironment returns the tracing environment or falls back to provided default
func (c *TracingConfig) GetEnvironment(fallback string) string {
	if c.Environment != "" {
		return c.Environment
	}
	return fallback
}

// GetServiceName returns the tracing service name or falls back to provided default
func (c *TracingConfig) GetServiceName(fallback string) string {
	if c.ServiceName != "" {
		return c.ServiceName
	}
	return fallback
}

// GetServiceVersion returns the tracing service version or falls back to provided default
func (c *TracingConfig) GetServiceVersion(fallback string) string {
	if c.ServiceVersion != "" {
		return c.ServiceVersion
	}
	return fallback
}

// InitializeTracing sets up OpenTelemetry tracing with Sentry.io integration
// Spec: docs/specs/004-opentelemetry-tracing.md#1-tracing-initialization-module
func InitializeTracing(cfg TracingConfig) (func(), error) {
	if !cfg.Enabled {
		// Set up no-op tracer provider
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
		return func() {}, nil
	}

	// Initialize Sentry with tracing enabled only if DSN is provided
	if cfg.SentryDSN != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.SentryDSN,
			Environment:      cfg.GetEnvironment("development"),
			Release:          cfg.GetServiceVersion("v1.0.0"),
			EnableTracing:    true,
			TracesSampleRate: cfg.SampleRate,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Sentry: %w", err)
		}
	}

	// Create resource with service identification
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.GetServiceName("unknown-service")),
		semconv.ServiceVersion(cfg.GetServiceVersion("v1.0.0")),
		semconv.DeploymentEnvironment(cfg.GetEnvironment("development")),
	)

	// Create tracer provider options
	tpOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SampleRate)),
	}
	
	// Add Sentry span processor only if DSN is provided
	if cfg.SentryDSN != "" {
		tpOpts = append(tpOpts, sdktrace.WithSpanProcessor(sentryotel.NewSentrySpanProcessor()))
	}
	
	// Create tracer provider
	tp := sdktrace.NewTracerProvider(tpOpts...)

	// Set global tracer provider
	otel.SetTracerProvider(tp)
	
	// Set propagator - include Sentry propagator only if DSN is provided
	var propagators []propagation.TextMapPropagator
	if cfg.SentryDSN != "" {
		propagators = append(propagators, sentryotel.NewSentryPropagator())
	}
	propagators = append(propagators, propagation.TraceContext{}, propagation.Baggage{})
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagators...))

	// Return cleanup function
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tp.Shutdown(ctx)
		if cfg.SentryDSN != "" {
			sentry.Flush(2 * time.Second)
		}
	}, nil
}