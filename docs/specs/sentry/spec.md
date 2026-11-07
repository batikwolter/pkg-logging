# Optional Sentry structured logs

## Approval and scope

User approved handler composition, dependency addition, implementation, and bounded agents on 2026-11-07. Baseline: e1ae7d6149cea72a609efd2f1ea3f8094e289ec9. Scope: this repository only. No bot changes, commits, publishing or deployment. One read-only design scout and one independent reviewer; no recursive delegation. Supervisor: s7l-super-logging, w8E:p2. Scout: s7l-child-design, w8F:p1, isolated linked worktree at baseline.

## Goal

Existing observe.Logger calls optionally emit Sentry structured logs while preserving JSON stdout and current schema, redaction, context, trace-generation policy and public logger methods. Sentry Issues remain explicitly captured by applications, not inferred from log severity.

## Design

- Retain private slog backend and existing redaction order. Internally compose JSON handler with optional destination handlers before binding resource fields.
- Add an opaque observe.Destination value (alias of internal/output.Destination) and Config.Destinations []Destination. Destination has no public methods exposing slog. Only in-module companion factories construct nonzero destinations. Zero destination is invalid configuration.
- internal/output owns the opaque value, factory and extraction functions. Its slog types are internal, not public observe contract. This is deliberately closed to third-party destinations; revisit only with concrete second integration need. No registration, dynamic discovery or generic plugin system.
- observe/sentry.New(client *sentry.Client) (observe.Destination, error) wraps official github.com/getsentry/sentry-go/slog v0.50.0 handler. Client must be nonnil. Caller owns client configuration, privacy callbacks, initialization and bounded shutdown flush. No SDK globals changed.
- Applications append returned destination to Config.Destinations, then use existing observe.New. Existing config literals and calls remain valid. Same module; Sentry dependencies enter module graph but core observe imports no SDK.
- Internal fan-out uses Config.Level as common minimum and checks each child Enabled. JSON attempted first; eligible remote handlers attempted even after JSON failure. Combine errors internally; existing void log methods retain existing inability to report handler errors. Do not recursively log failures.
- WithAttrs/WithGroup derive independent child handlers; clone record per child. Destination handlers see field-redacted resource, bound and call attributes. Existing duplicate-key JSON behavior stays unchanged; official SDK determines remote duplicate resolution.
- Sentry companion binds supplied client through private hub fallback while preserving per-call context. Use request hub only when bound to supplied client; otherwise use fresh scope with supplied client, preventing accidental cross-client routing. No request scope mutated. Existing custom trace.id remains ordinary attribute; do not synthesize SDK trace identity.
- SDK buffering owns network batching. No custom queue, retries, per-log flush, panic recovery or delivery durability guarantee. Handler failures cannot suppress independent stdout attempt.
- Keep Go 1.25 baseline; implement small internal fan-out rather than relying on Go 1.26 slog.NewMultiHandler.

## Alternatives and evidence

Current observe.New hardcodes JSON handler. Resource and With fields already live in slog handler state. External wrappers duplicate redaction/level/bound-field behavior and lose private trace policy. JSON writer bridges lose per-call context. A public Record/Sink interface introduces duplicate record model and conversion; prefer opaque destinations reusing native handlers. Design scout recommended Record/Sink but explicitly identified internal construction as alternative; opaque destination is narrower than public slog bridge or generic sink.

Source: observe/logger.go, config.go, fields.go, redact.go, context.go, grpc/server.go, grpc/client.go; INTEGRATION.md; docs/init.md. Existing non-goal against exporter frameworks remains: one optional companion uses SDK, core adds destination composition only.

## Privacy and limits

Existing redaction covers top-level sensitive field keys/flags, not message text, error text, nested values or arbitrary objects. Do not claim stronger guarantees. Sentry SDK can enrich logs from scope; caller must set safe scope and BeforeSendLog/BeforeSend callbacks. No default PII opt-in. Sentry failure does not fail RPC, and buffered remote records may be lost on crashes/outages. Official handler may use emission timestamp rather than original slog timestamp; no exact remote timestamp/source parity promise.

## Acceptance

1. Existing New configuration and JSON tests pass unchanged; no slog appears in public observe signatures.
2. Global level filtering applies to all destinations; child eligibility honored.
3. Resource, chained With and per-call fields reach both destinations; parent/sibling binding isolated.
4. Sensitive fields masked at resource/With/call binding before both destinations.
5. JSON and secondary outputs independently attempted on handler errors; records isolated from child mutation.
6. Concurrent logs and With calls pass race detector.
7. Disabled trace generation preserved through destination configuration and With; interceptors still emit one completion log per unary RPC per destination.
8. Nil Sentry client and zero destination rejected. SDK integration tests use local fake transport, flush buffered logs, assert structured logs not exception events, supplied client routing and context/scope behavior. No external Sentry calls.
9. README, INTEGRATION and changelog document opt-in setup, lifecycle, privacy limits and non-guaranteed delivery.

## Delivery and checks

Implement internal output composition and tests; add Sentry companion with pinned SDK and fake-transport tests; update docs. Run go test ./..., go test -race ./..., go vet ./..., and Go 1.25 compatibility check if installed/available. rtk absent in current environment, so run native Go commands and record deviation.

Rollback: omit Destinations; default JSON behavior unchanged. No migration or deployed changes.

## Review and evidence

Implemented and verified on 2026-11-07. Native go test ./..., go test -race ./..., go vet ./..., GOTOOLCHAIN=go1.25.0 go test -count=1 ./..., and git diff --check passed. rtk unavailable; native Go commands used. Fake transport tests made no external Sentry calls. No live Sentry delivery or bot rollout tested.

Independent reviewer s7l-child-review, w8G:p1, branch s7l-child-review, linked worktree /Users/wolter/.herdr/worktrees/pkg-logging/s7l-child-review reviewed baseline plus full tracked/untracked snapshot c4269e3d7a63e2e7b23d841c02e10748a4983183edb83a3b88354dd098e257ce. Source-only security/intent/sufficiency/simplicity/project-direction review accepted with two nonblocking test gaps: shared-record pressure and unary-client destination coverage. Both fixed: added internal/output/isolation_test.go and TestClientCompletionOncePerDestination. Mutation experiment removing r.Clone() failed isolation regression; restoring clone passed. All checks rerun successfully. Same reviewer rechecked both updated files and production interactions, accepted, no blockers. Reviewer made no edits or test runs; full OS sandbox unavailable, so independent runtime security validation not performed. Scout and reviewer worktrees retained.

Residual limits: explicit unkeyed Config struct literals need field-count updates after additive field (ordinary Go struct-extension limitation); remote SDK dependency graph is shared; no exact timestamp/source parity or durable shipping; no bot exception-capture integration. Existing documented privacy limits remain. No commits/push/deployment performed.
