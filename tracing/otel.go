// Package tracing provides OpenTelemetry integration.
package tracing

import (
  "context"
  "fmt"

  "go.opentelemetry.io/otel"
  "go.opentelemetry.io/otel/attribute"
  "go.opentelemetry.io/otel/exporters/otlp/otlptrace"
  "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
  "go.opentelemetry.io/otel/sdk/resource"
  sdktrace "go.opentelemetry.io/otel/sdk/trace"
  oteltrace "go.opentelemetry.io/otel/trace"
)

var (
	tracerProvider *sdktrace.TracerProvider
	tracer         oteltrace.Tracer
)

// SetupOTEL configures OpenTelemetry with the given endpoint and service name.
// If endpoint is empty, no telemetry is exported.
func SetupOTEL(endpoint, serviceName string) error {
	// No-op if endpoint is not configured
	if endpoint == "" {
		return nil
	}

// Create OTLP gRPC exporter
	client := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(endpoint),
	)
	exporter, err := otlptrace.New(context.Background(), client)
	if err != nil {
		return fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create tracer provider with resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
			attribute.String("service.version", "1.0.0"),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider with immediate export (SimpleSpanProcessor)
	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)),
		sdktrace.WithResource(res),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tracerProvider)
	tracer = otel.Tracer("bantam")

	return nil
}

// StartSpan starts a new span with the given name and attributes.
func StartSpan(ctx context.Context, name string, attrs map[string]string) context.Context {
	if tracerProvider == nil {
		return ctx
	}

	spanAttrs := make([]oteltrace.SpanStartOption, 0, len(attrs))
	for k, v := range attrs {
		spanAttrs = append(spanAttrs, oteltrace.WithAttributes(attribute.String(k, v)))
	}

	ctx, span := tracer.Start(ctx, name, spanAttrs...)
	span.End()

	return ctx
}

// StartActiveSpan starts a span and returns both context and span for later ending.
func StartActiveSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, oteltrace.Span) {
	if tracerProvider == nil {
		return ctx, nil
	}

	spanAttrs := make([]oteltrace.SpanStartOption, 0, len(attrs))
	for k, v := range attrs {
		spanAttrs = append(spanAttrs, oteltrace.WithAttributes(attribute.String(k, v)))
	}

	return tracer.Start(ctx, name, spanAttrs...)
}

// EndSpan ends the given span.
func EndSpan(span oteltrace.Span) {
	if span != nil {
		span.End()
	}
}

// ShutdownOTEL shuts down the tracer provider.
func ShutdownOTEL() error {
	if tracerProvider == nil {
		return nil
	}
	return tracerProvider.Shutdown(context.Background())
}

// GetTracer returns the global tracer.
func GetTracer() oteltrace.Tracer {
	return tracer
}
