You are a senior Go platform engineer.

Build a **production-ready shared Go logging package** for internal service use.

This package is intended to become the organization-wide standard for structured logging in Go services, especially gRPC services. The design should be stable, low-maintenance, easy to adopt, and future-compatible without being over-engineered.

Do **not** build a full observability platform. Do **not** build OTel, Prometheus, exporters, or dynamic config systems yet.

The goal is to implement a **foundation-ready logging core** that can later evolve into a broader observability package without breaking service code.

## Repository goal

Implement a repository with this structure:

```text
observe/
  logger.go
  config.go
  resource.go
  context.go
  fields.go
  event.go
  error.go
  redact.go
  testkit/
    sink.go
    assert.go
  grpc/
    server.go
    client.go
    metadata.go
```

Use one Go module. Keep the package small and coherent.

## Core requirements

Implement a shared package named `observe`.

This package must provide:

* structured JSON logging
* a small org-owned logger interface
* config with sane defaults
* resource identity metadata
* logger propagation through `context.Context`
* standard field helpers
* standard event naming helpers/constants
* error classification helpers
* minimal redaction hooks
* gRPC unary server interceptor
* gRPC unary client interceptor
* test utilities for validating emitted logs

The package should write JSON logs to stdout/stderr.
It should not implement log shipping, collector/exporter logic, tracing SDKs, metrics SDKs, or dashboards.

## Design rules

Follow these rules carefully:

1. Keep the public API small.
2. Use Go `slog` internally if practical, but do not expose `slog` directly as the main public contract.
3. Prefer a thin org-owned wrapper over direct backend exposure.
4. Use flat dotted log field names.
5. Use one JSON log event per line.
6. Keep operational/request logs separate from domain/business logs.
7. Let interceptors own request lifecycle logs.
8. Let service handlers own domain/business logs.
9. Make safe defaults easy.
10. Avoid giant abstraction hierarchies.

## Public API target

Implement something close to this:

```go
package observe

type Logger interface {
    DebugContext(ctx context.Context, msg string, fields ...Field)
    InfoContext(ctx context.Context, msg string, fields ...Field)
    WarnContext(ctx context.Context, msg string, fields ...Field)
    ErrorContext(ctx context.Context, msg string, fields ...Field)

    With(fields ...Field) Logger
}

type Field struct {
    Key        string
    Value      any
    Sensitive  bool
    MetricSafe bool
}

type Resource struct {
    ServiceName    string
    ServiceVersion string
    Environment    string
}

type Config struct {
    Resource  Resource
    Level     string
    Format    string
    Output    string
    AddSource bool
}

func New(cfg Config) (Logger, error)
func FromContext(ctx context.Context) Logger
func IntoContext(ctx context.Context, logger Logger) context.Context
```

You may refine these types slightly if needed, but keep the API small and stable.

## File responsibilities

### `logger.go`

Implement:

* logger interface
* default logger implementation
* JSON output setup
* `With(fields...)`
* context-aware logging methods
* sane fallbacks

### `config.go`

Implement:

* config model
* validation
* defaults
* optional env-based config helpers if useful

Defaults:

* JSON format
* stdout output
* info level
* source disabled by default

### `resource.go`

Implement:

* `Resource` type
* resource field injection into every log line
* clear separation of service identity metadata from request metadata

### `context.go`

Implement:

* storing/retrieving logger from context
* storing/retrieving request-scoped metadata from context if needed
* safe fallback behavior when no logger is present
* helpers must never panic

### `fields.go`

Implement:

* `Field` type
* field constructors:

  * `String`
  * `Int`
  * `Bool`
  * `Float64`
  * `Duration`
  * `Time`
  * `Error`
  * `SensitiveString`
* conversion helpers for internal logger backend
* keep room for future field governance

### `event.go`

Implement:

* standard event constants/helpers for operational logs
* examples:

  * `rpc.server.completed`
  * `rpc.client.completed`
* keep event naming stable and dot-separated

### `error.go`

Implement:

* error classification helpers
* mapping helpers for gRPC status codes
* helper fields such as:

  * `error.message`
  * `error.code`
  * `error.kind`
* support distinction between business errors and unexpected/internal errors where practical
* do not turn this into a giant general-purpose error framework

### `redact.go`

Implement:

* minimal but useful redaction support
* denylist-based masking/drop for obvious sensitive keys:

  * authorization
  * cookie
  * password
  * secret
  * token
  * api key
* final safety pass for emitted fields
* keep implementation intentionally simple
* do not build a full PII detection engine

### `grpc/metadata.go`

Implement:

* metadata normalization helpers
* extraction of:

  * `x-request-id`
  * `x-correlation-id`
  * `x-trace-id`
  * `x-source-service`
* outbound propagation helpers
* fallback generation rules
* be careful that gRPC metadata keys are effectively lowercase

Rules:

* if `request.id` is missing, generate one
* if `correlation.id` is missing, it may default to request ID
* preserve existing values when present
* do not duplicate outbound metadata carelessly

### `grpc/server.go`

Implement unary server interceptor that:

1. captures start time
2. extracts inbound metadata
3. enriches logger with request metadata
4. injects logger into context
5. calls handler
6. classifies outcome
7. emits one completion log per request

Do not emit both start and completion logs in v1.

Completion log should include fields like:

* `event=rpc.server.completed`
* `service.name`
* `service.version`
* `env`
* `rpc.system=grpc`
* `grpc.service`
* `grpc.method`
* `grpc.status_code`
* `duration_ms`
* `request.id`
* `correlation.id`
* `trace.id`
* `outcome`
* `peer.address` when practical

### `grpc/client.go`

Implement unary client interceptor that:

1. reads logger/request metadata from context
2. ensures outbound metadata exists
3. propagates request/correlation/trace/source-service fields
4. measures outbound call duration
5. emits one completion log per client RPC

Completion log should include fields like:

* `event=rpc.client.completed`
* `target.service`
* `grpc.method`
* `grpc.status_code`
* `duration_ms`
* `request.id`
* `correlation.id`
* `trace.id`
* `outcome`

## Schema rules

Use flat dotted fields consistently.

Examples:

* `service.name`
* `service.version`
* `request.id`
* `correlation.id`
* `trace.id`
* `grpc.method`
* `grpc.status_code`
* `error.code`
* `error.kind`

Do not mix styles like `request_id`, `requestId`, and `request.id`.

Use stable, lowercase, dot-separated event names.

Examples:

* `rpc.server.completed`
* `rpc.client.completed`
* `order.approved`
* `order.validation.failed`

## Logging rules

Implement and document these rules:

### Operational/request logs

* emitted by interceptors
* one completion log per unary RPC in v1
* no request/response body logging by default

### Domain/business logs

* emitted by service handlers
* include relevant business identifiers
* use event names explicitly
* do not duplicate automatic request completion logs

### Error logs

* expected business failures may be `WARN`
* unexpected/system failures should be `ERROR`
* interceptor logs request outcome
* handler may log domain meaning, but avoid duplicate full error stories

## Test support

Implement `testkit/` with:

* in-memory sink for captured structured log entries
* assertion helpers

Add tests for:

* logger creation
* context fallback behavior
* field conversion
* redaction behavior
* gRPC metadata propagation
* server interceptor completion log behavior
* client interceptor propagation behavior

At least one integration-style test should verify:

* request metadata enters server interceptor
* logger in handler context contains the expected IDs
* completion log is emitted with expected schema fields

## Example usage

Include a small runnable example or README snippet showing:

```go
cfg := observe.Config{
    Resource: observe.Resource{
        ServiceName:    "order-service",
        ServiceVersion: "1.0.0",
        Environment:    "prod",
    },
    Level:  "info",
    Format: "json",
    Output: "stdout",
}

logger, err := observe.New(cfg)
if err != nil {
    panic(err)
}

server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        observegrpc.UnaryServerInterceptor(logger),
    ),
)
```

And a handler example using:

```go
log := observe.FromContext(ctx)
log.InfoContext(ctx, "order approved",
    observe.String("event", "order.approved"),
    observe.String("order.id", orderID),
)
```

## Implementation quality bar

The result must be:

* idiomatic Go
* practical and maintainable
* small in public API
* well-tested
* future-compatible
* not over-engineered

Do not add:

* HTTP middleware
* worker middleware
* OTel SDK integration
* Prometheus support
* exporter modules
* dynamic remote config
* complex plugin systems

## Output format

Produce:

1. full repository tree
2. all important source files with real code
3. `go.mod`
4. tests
5. concise README or setup notes
6. brief explanation of tradeoffs

Choose the most practical implementation when tradeoffs appear.

Prioritize:

* stability
* clarity
* low adoption friction
* schema consistency
* safe defaults
