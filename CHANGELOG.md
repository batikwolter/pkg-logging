# Changelog

All notable changes to this project will be documented in this file.

Format follows Keep a Changelog principles and semantic versioning.

## [0.3.0] - 2026-11-08

### Added
- Optional `observe/otlp.New(log.LoggerProvider)` destination using official `otelslog v0.20.0`; nil and typed-nil providers rejected, no global provider selection.
- Conversion, binding, duplicate resolution, redaction, native context correlation, isolation and concurrent logging regressions.
- OTLP ownership/privacy integration guidance alongside stdout and Sentry.

### Changed
- Shared module includes OTel log API/SDK v0.21.0 and stable OTel v1.45.0; Go 1.25 baseline preserved. SDK used only by companion tests; core imports no OTel SDK.

## [0.2.0] - 2026-11-07

### Added
- Optional opaque `Config.Destinations` with internal handler composition; JSON output remains enabled.
- `observe/sentry.New` companion using Sentry Go SDK v0.50.0 for structured logs, with app-owned client lifecycle and exception capture.
- Destination isolation, redaction, concurrency, client-routing and unary completion regression tests.
- MIT license for public package consumption.
- CI workflow for `push` and `pull_request` validation with `go test ./...`.

### Changed
- Documented public-use support posture in `README.md`.

## [0.1.1] - 2026-04-21

### Added
- AGENTS project playbook with fast context map and stable contract rules.
- `docs/releasing.md` release checklist and versioning workflow.
- `scripts/release.sh` helper for guarded local release flow (check/test/commit/tag/push).
- `.github/workflows/release-tag.yml` to validate and publish GitHub release on semver tag push.

### Changed
- Consolidated integration guidance into root `INTEGRATION.md` as single source of truth.

## [0.1.0] - 2026-04-20

### Added
- `observe` package with org-owned logger interface over JSON structured logging.
- Config with sane defaults (`json`, `stdout`, `info`, source disabled by default).
- Resource identity fields (`service.name`, `service.version`, `env`).
- Context helpers for logger and request metadata propagation.
- Flat dotted field helpers and stable event helpers.
- Error classification helpers with gRPC status mapping.
- Redaction denylist safety pass for sensitive keys.
- gRPC unary server interceptor with one completion log per request.
- gRPC unary client interceptor with metadata propagation and completion logs.
- `observe/testkit` in-memory sink and assertion helpers.
- Integration documentation and README quick start.
