# INTEGRATION

Standard integration contract for AI agents adopting `github.com/deccara-tech/pkg-logging`.

## 1) Goal
Use this package to produce consistent, structured, safe logs across Go gRPC services.

Rules:
- keep API usage simple
- keep schema stable
- avoid duplicate logging across layers

Install:
```bash
go get github.com/deccara-tech/pkg-logging@latest
```

## 2) Quick Integration
1. Create one startup logger with `observe.New(...)`.
2. Attach `observegrpc.UnaryServerInterceptor(logger)` on gRPC servers.
3. Attach `observegrpc.UnaryClientInterceptor(logger, observegrpc.WithSourceService(...))` on outbound gRPC clients.
4. In handlers, always use `observe.FromContext(ctx)`.

## 3) Minimum Log Contract
Every meaningful log should include:
- `event`
- `service.name` (equivalent of service_name)
- `env` (equivalent of environment)
- `correlation.id`

Auto fields from backend/interceptors include timestamp/level and request lifecycle context.

## 4) Event Naming
Use:
- `component.action.outcome`

Examples:
- `order.create.started`
- `order.create.success`
- `order.create.failed`
- `external.payment_api.request.success`

Operational events reserved by package:
- `rpc.server.completed`
- `rpc.client.completed`

## 5) Ownership by Layer
- Interceptors: request lifecycle completion logs.
- Handlers/business logic: domain events.

Do not duplicate the same failure story in multiple layers.

## 6) Context Propagation
Across boundaries, propagate:
- `x-request-id`
- `x-correlation-id`
- `x-trace-id`
- `x-source-service`

Behavior:
- preserve existing values
- generate `request.id` if missing
- default `correlation.id` to request ID if missing
- `trace.id` generation controlled by `Config.GenerateTraceID` (default: true)

## 7) Sensitive Data Rules
Never log secrets or personal sensitive payloads.

Blocked examples:
- password
- token / api key
- cookie
- auth headers
- payment secrets

Package redaction denylist is active; still prefer logging only safe fields.

## 8) External API Logging
When calling external dependencies, log:
- dependency/service name
- action/method
- status code/result
- `duration_ms`
- safe identifiers only

Do not log raw request/response bodies by default.

## 9) Log Levels
- DEBUG: detailed diagnostics
- INFO: business milestones
- WARN: recoverable anomalies / expected business failures
- ERROR: failed operations / unexpected failures

If you need CRITICAL semantics, emit `ErrorContext` with explicit event and severity field (for example `severity=critical`).

## 10) Error Logging
For failures, include context:
- `event`
- `error.code`
- `error.kind`
- `error.message`
- sanitized business identifiers

## 11) Performance Logging
Use performance fields where meaningful:
- `duration_ms`
- `retry_count`
- `unit_count` (batch size/records)

Avoid high-noise micro logs for trivial operations.

## 12) Audit vs Operational
- Operational logs: runtime behavior/debug/performance.
- Audit logs: actor/action business accountability.

For audit entries, add:
- `log.type=audit`
- actor id
- action/event
- timestamp (auto)

## 13) Volume and Stability
- avoid loop spam and repeated identical logs
- summarize high-volume operations
- keep one JSON event per line
- keep dotted flat keys only (no snake_case/camelCase variants)

## 14) Validation Gate
Before merge:
```bash
rtk go test ./...
```

If logging behavior/schema changed, update tests under:
- `observe/logger_test.go`
- `observe/grpc/*_test.go`

## 15) Non-Goals
Do not expand this package into:
- full observability platform
- metrics/tracing SDK integration in core `observe` (optional `observe/otlp` uses official log API/bridge; applications own SDK/exporters/tracing)
- exporter/collector framework
- dynamic remote config system

## 17) Optional Sentry Logs

`observe/sentry` forwards structured logs using the official Sentry slog handler.
JSON stdout remains enabled. Existing log calls do not change.

```go
import (
    "context"
    "time"

    "github.com/deccara-tech/pkg-logging/observe"
    observesentry "github.com/deccara-tech/pkg-logging/observe/sentry"
    "github.com/getsentry/sentry-go"
)

func run(ctx context.Context, dsn string) error {
    client, err := sentry.NewClient(sentry.ClientOptions{
        Dsn: dsn,
        Environment: "prod",
        Release: "audit-bot@1.0.0",
        SendDefaultPII: false,
        // Supply BeforeSendLog and BeforeSend for application-specific scrubbing.
    })
    if err != nil { return err }
    defer client.Close()
    defer client.Flush(2 * time.Second) // runs before Close; check result in production

    destination, err := observesentry.New(client)
    if err != nil { return err }
    logger, err := observe.New(observe.Config{
        Resource: observe.Resource{ServiceName: "audit-bot", Environment: "prod"},
        Destinations: []observe.Destination{destination},
    })
    if err != nil { return err }
    logger.InfoContext(ctx, "bot ready", observe.Event("bot.ready"))
    // Run service here; return after workers stop, before deferred flush.
    return nil
}
```

- SDK versions are pinned in `go.mod`; v0.50.0 enables logs by default.
- `Config.Level` is the common minimum for stdout and Sentry. Use `BeforeSendLog`
  for additional remote filtering, including event-based selection of INFO job failures.
- Request/job hubs bound to the supplied client preserve SDK scope. Missing hubs or
  hubs for another client use a fresh scope with the supplied client; no globals change.
- `trace.id` stays a searchable custom field, not automatically a native Sentry trace.
- This destination emits Logs only. Capture unexpected errors/panics explicitly at
  one application boundary for Issues; `observe.Error` contains text, not original error.
- Top-level field masking applies before either destination, including resource and
  `With` fields. It does not scrub message/error text, nested objects or SDK-enriched
  scope data. Keep safe scope data and configure `BeforeSendLog` / `BeforeSend`.
  `SendDefaultPII: false` alone does not remove explicitly supplied personal data.
- SDK buffering is best-effort, not durable delivery. No custom queue or retries.
  Output errors do not skip the other output; existing void log methods do not return them.
  Blocking writers/handlers still block the call; independence is not timeout isolation.
- Flush only after service tasks finish. `os.Exit` and crashes bypass deferred flush.
  Check `client.Flush` result during real shutdown. Remote timestamp/source metadata
  may differ from stdout; Sentry resolves duplicate attributes using SDK behavior.
- Omit `Destinations` to restore existing JSON-only behavior. Zero destinations and
  nil clients are rejected. No generic destination registration API exists.

## 18) Optional OTLP Logs

`observe/otlp` (package `observeotlp`) converts logs through official `otelslog`.
It takes an explicit `go.opentelemetry.io/otel/log.LoggerProvider`:

```go
// provider is application-owned and already configured with a fail-closed
// privacy wrapper around a bounded SDK batch processor. Never attach an
// unfiltered exporter to a provider receiving arbitrary application logs.
destination, err := observeotlp.New(provider)
if err != nil { return err }
logger, err := observe.New(observe.Config{
    Resource: observe.Resource{ServiceName: "audit-bot", Environment: "prod"},
    Destinations: []observe.Destination{destination}, // append Sentry destination here
})
if err != nil { return err }
logger.InfoContext(ctx, "job ready", observe.Event("job.ready"))
```

- Nil and typed-nil providers return an error and invalid destination. No global
  provider is selected or changed. No exporter, network client, resources, tracing,
  environment configuration, flush or shutdown is created by this companion.
- Pins: `otelslog v0.20.0`, log API/SDK `v0.21.0`, stable OTel `v1.45.0`.
  Root module remains Go 1.25. SDK dependencies support companion tests; core
  `observe` imports no OTel SDK. HTTP exporters are not package dependencies.
  Compatible bot pins: `otlploghttp v0.21.0`, `otlptracehttp v1.45.0`.
- `Config.Level` remains common minimum. Provider `Enabled` may further filter
  OTLP without dropping stdout or Sentry. JSON schema and gRPC completion behavior
  stay unchanged. SDK configuration can still block logging; companion adds no
  asynchronous isolation itself.
- Message becomes body; timestamp and standard DEBUG/INFO/WARN/ERROR severity are
  preserved. Resource/With/call fields become log attributes. SDK default duplicate
  resolution is last value wins, including call values over With/resource fields.
  Do not enable duplicate preservation without resolving duplicates before policy.
- Valid active OTel context supplies native TraceId/SpanId through SDK. Legacy
  `trace.id`, `request.id`, `correlation.id` attributes do not create native parents.
  Companion adds no source attributes; scope name is fixed to
  `github.com/deccara-tech/pkg-logging/observe/otlp`.

### Privacy and lifecycle ownership

Core masking covers top-level field keys only. Messages, raw errors, nested objects,
SDK enrichment and provider Resource are not made safe by this destination.

Register **one privacy wrapper around downstream batch processor**, not separate
privacy and batch processors. SDK invokes every registered processor even when
an earlier `OnEmit` returns an error. Unknown/missing/wrong-type events must return
without invoking downstream; clearing body or returning an error does not drop a
record from separately registered exporters. Wrapper must re-evaluate each record,
rebuild body/attributes from closed approved vocabulary, reject nested/unknown data,
and enforce finite numeric and payload bounds. Delegate `Enabled`, `ForceFlush`
and `Shutdown` to wrapped processor; keep processing race-safe. Provider Resource
and instrumentation scope need independent trusted allowlists. Register downstream
only through wrapper, exactly once. Keep diagnostics local, outside remote fan-out.

Bot-owned starting bounds supported by selected SDK: `WithMaxQueueSize(512)`,
`WithExportMaxBatchSize(128)`, `WithExportInterval(time.Second)` and
`WithExportTimeout(time.Second)`. Configure exporter timeout/retry budget and
transport bounds separately; queues are best-effort and drop on saturation.
No per-record flush, synchronous exporter or durable-delivery promise.

Apps own trusted endpoint/header validation, native propagation and trace privacy.
Shutdown after intake/workers stop, freeze late telemetry, then share one bounded
fresh-context budget across OTel provider shutdowns and Sentry flush/close.
Companion never owns those lifecycle calls. Omit destination to disable OTLP.

## 16) Expected Outcome
After adoption, services should get:
- one consistent JSON logging contract across repos
- automatic unary gRPC completion logs (server/client)
- stable request correlation propagation across boundaries
- safer logs with redaction defaults
- clearer split between operational logs and domain logs
