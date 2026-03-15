# OpenTelemetry Implementation Plan for Bantam

## Current State
- OpenTelemetry SDK is imported and configured
- Console exporter outputs JSON spans to stderr
- Spans created for: `process_message`, `llm.chat`, `llm.provider_call`, `tool.execute`
- Traces not being sent to configured endpoint yet

## Goal
Send OpenTelemetry traces to the configured gRPC endpoint from config file

## Configuration
The config file at `~/.bantam/config.yaml` has:
```yaml
tracing:
  endpoint: ""  # Empty = debug mode (console stderr)
                # Set to gRPC endpoint like localhost:4317
  serviceName: bantam
```

## Implementation Steps

### 1. Update tracing/otel.go
- Add OTLP gRPC exporter import: `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc`
- Replace `stdouttrace.New` with conditional logic:
  - If `endpoint == ""`: use `stdouttrace.New` with `SimpleSpanProcessor` (debug mode)
  - If `endpoint != ""`: use `otlptracegrpc.NewClient` with `BatchSpanProcessor` (production mode)
- Remove debug `fmt.Fprintf` calls
- Remove `Flush()` call from `EndSpan()`

### 2. Config file format (already exists)
```yaml
tracing:
  endpoint: "localhost:4317"  # gRPC endpoint
  serviceName: "bantam"
```

### 3. Testing
- Empty endpoint: console output to stderr (debug mode)
- Set endpoint: send traces to gRPC endpoint (production mode)

## Files to Modify
- `/home/lore/.nanobot/workspace/src/bantam/tracing/otel.go`
