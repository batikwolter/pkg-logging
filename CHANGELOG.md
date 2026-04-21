# Changelog

All notable changes to this project will be documented in this file.

Format follows Keep a Changelog principles and semantic versioning.

## [Unreleased]

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
