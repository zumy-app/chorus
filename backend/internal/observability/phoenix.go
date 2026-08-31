// Package observability wires OpenTelemetry tracing to Arize Phoenix (NFR-25).
//
// Phoenix is an OpenTelemetry-compatible collector that stores traces, datasets,
// and prompt versions for offline + realtime evaluation of translation/grammar.
// The backend exports spans from TranslationService and GrammarService over OTLP
// gRPC. It is deliberately opt-in and off the request hot path: when PHOENIX is
// not enabled (or unreachable) exported spans are dropped synchronously and the
// producer path is untouched.
package observability

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const (
	// PhoenixEnabledEnv toggles tracing on/off.
	PhoenixEnabledEnv = "PHOENIX_ENABLED"
	// PhoenixOTLPEndpointEnv is the OTLP gRPC endpoint (e.g. "phoenix:4317").
	PhoenixOTLPEndpointEnv = "PHOENIX_OTLP_ENDPOINT"
	// PhoenixServiceNameEnv overrides the service name shown in Phoenix.
	PhoenixServiceNameEnv = "PHOENIX_SERVICE_NAME"
	// PhoenixSampleRateEnv controls the trace sample rate (0.0-1.0).
	PhoenixSampleRateEnv = "PHOENIX_SAMPLE_RATE"

	defaultOTLPEndpoint = "http://localhost:4317"
	defaultServiceName  = "chorus-backend"
	defaultSampleRate   = 1.0
)

// PhoenixEnabled reports whether tracing has been enabled via PHOENIX_ENABLED.
func PhoenixEnabled() bool {
	v := os.Getenv(PhoenixEnabledEnv)
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// SetupPhoenix configures the global TracerProvider to export spans to a local
// Arize Phoenix instance over OTLP gRPC. It returns a shutdown function that
// flushes and stops the exporter (call it on graceful shutdown).
//
// When Phoenix is disabled it returns a no-op shutdown and leaves the global
// provider at the default no-op so tracing adds zero overhead.
func SetupPhoenix() (shutdown func(context.Context) error) {
	if !PhoenixEnabled() {
		return func(context.Context) error { return nil }
	}

	endpoint := os.Getenv(PhoenixOTLPEndpointEnv)
	if endpoint == "" {
		endpoint = defaultOTLPEndpoint
	}
	serviceName := os.Getenv(PhoenixServiceNameEnv)
	if serviceName == "" {
		serviceName = defaultServiceName
	}
	sampleRate := envSampleRate()

	exporter, err := otlptracegrpc.New(context.Background(),
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		log.Printf("[Phoenix] failed to create OTLP exporter (tracing disabled): %v", err)
		return func(context.Context) error { return nil }
	}

	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		"chorus",
		attribute.String("service.name", serviceName),
		attribute.String("service.version", "2.0.0"),
		attribute.String("deployment.environment", os.Getenv("ENVIRONMENT")),
	))
	if err != nil {
		res = resource.Default()
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleRate))),
	)
	otel.SetTracerProvider(tp)

	log.Printf("[Phoenix] tracing enabled -> OTLP gRPC %s (service=%s, sample=%.2f)", endpoint, serviceName, sampleRate)
	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			return err
		}
		return nil
	}
}

// Tracer returns a named tracer for a subsystem (e.g. "translation", "grammar").
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// envSampleRate parses PHOENIX_SAMPLE_RATE into [0,1].
func envSampleRate() float64 {
	if v := os.Getenv(PhoenixSampleRateEnv); v != "" {
		if r, err := strconv.ParseFloat(v, 64); err == nil && r >= 0 && r <= 1 {
			return r
		}
	}
	return defaultSampleRate
}
