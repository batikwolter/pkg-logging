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
- metrics/tracing SDK integration
- exporter/collector framework
- dynamic remote config system

## 16) Expected Outcome
After adoption, services should get:
- one consistent JSON logging contract across repos
- automatic unary gRPC completion logs (server/client)
- stable request correlation propagation across boundaries
- safer logs with redaction defaults
- clearer split between operational logs and domain logs
