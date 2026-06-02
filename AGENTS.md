@INTEGRATION.md
@docs/releasing.md

# AGENTS: pkg-logging

## Mission
Build and maintain a small, stable, org-owned Go logging foundation for gRPC services.

Primary goals:
- stable API
- low adoption friction
- consistent JSON schema
- safe defaults

Non-goals:
- no OTel SDK
- no Prometheus
- no exporter/collector stack
- no dynamic remote config
- no plugin system

## Fast Context (Read This First)
If task is feature/bug/doc change, read only:
1. `README.md`
2. `INTEGRATION.md`
3. `observe/logger.go`
4. `observe/grpc/server.go`
5. `observe/grpc/client.go`

Only expand to other files if needed.

## Module + Packages
- Module: `github.com/deccara-tech/pkg-logging`
- Public package: `observe`
- Support packages:
  - `observe/grpc`
  - `observe/testkit`

## Stable Public API (Must Preserve)
From `observe`:
- `type Logger`
- `type Field`
- `type Resource`
- `type Config`
- `func New(cfg Config) (Logger, error)`
- `func FromContext(ctx context.Context) Logger`
- `func IntoContext(ctx context.Context, logger Logger) context.Context`
- field helpers (`String`, `Int`, `Bool`, `Float64`, `Duration`, `Time`, `Error`, `SensitiveString`)
- `BoolPtr(bool) *bool`

From `observe/grpc`:
- `UnaryServerInterceptor(logger observe.Logger)`
- `UnaryClientInterceptor(logger observe.Logger, opts ...ClientOption)`
- `WithSourceService(name string)`

Do not expose `slog` in public contract.

## Schema Contract
Use flat dotted keys only.

Required style:
- `service.name`
- `service.version`
- `env`
- `request.id`
- `correlation.id`
- `trace.id`
- `grpc.method`
- `grpc.status_code`
- `error.code`
- `error.kind`

Never add mixed styles (`snake_case`, `camelCase`) for same semantic field.

## Event Contract
Operational events:
- `rpc.server.completed`
- `rpc.client.completed`

Domain events:
- explicit, lowercase, dot-separated (`order.approved`, `order.validation.failed`)

## Logging Behavior Contract
- Interceptors own request lifecycle completion logs.
- Handlers own domain/business logs.
- No request/response body logging by default.
- One completion log per unary RPC in v1.

Severity guidance:
- expected business failure => `WARN`
- unexpected/system failure => `ERROR`

## Metadata + Trace Policy
Headers:
- `x-request-id`
- `x-correlation-id`
- `x-trace-id`
- `x-source-service`

Rules:
- preserve inbound/outbound values when present
- if `request.id` missing: generate
- if `correlation.id` missing: default to request ID
- `trace.id` fallback generation controlled by `Config.GenerateTraceID` (default true)

## Redaction Contract
Denylist-based masking for keys containing:
- `authorization`
- `cookie`
- `password`
- `secret`
- `token`
- `api key`

Keep redaction simple and deterministic.

## File Ownership Map
- `observe/logger.go`: core wrapper over slog + fallback logger + trace generation policy access
- `observe/config.go`: config defaults/validation + policy flags
- `observe/context.go`: logger + metadata context helpers
- `observe/fields.go`: field model + constructors
- `observe/error.go`: error classification + gRPC code mapping
- `observe/redact.go`: final masking pass
- `observe/grpc/metadata.go`: metadata extract/inject/normalize
- `observe/grpc/server.go`: server completion log interceptor
- `observe/grpc/client.go`: client propagation + completion log interceptor
- `observe/testkit/*`: sink + assertions for tests

## Testing Rules
Before merge:
```bash
rtk go test ./...
```

When touching schema/interceptors/redaction, add/adjust tests in:
- `observe/logger_test.go`
- `observe/grpc/*_test.go`

## Release Context
Current released tag baseline: `v0.1.0`.

For next releases, follow `docs/releasing.md` and update `CHANGELOG.md`.
Preferred command: `scripts/release.sh vX.Y.Z`.
Tag push automation: `.github/workflows/release-tag.yml`.

## Change Strategy
Prefer minimal diff.

Order of work:
1. preserve public API and schema stability
2. implement change
3. add/adjust tests
4. update docs only where contract changed

Avoid broad refactors unless explicitly requested.
