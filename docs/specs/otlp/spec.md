# Optional OTLP logs and bot tracing alongside Sentry

## Status and authority

Release follow-up: user explicitly authorized pkg-logging v0.3.0 commit/push/tag/publication, public module verification, and scoped bot integration/pin commits/pushes. No bot tags/releases, deployment, live telemetry or server changes authorized. Shipping's existing module replacement must also pin v0.3.0. Preserve unrelated work; never stage scratch/private files.

User selected direct Monoscope logs plus native traces alongside existing Sentry Logs/Issues and shared-module approach. On 2026-11-08 user approved implementation and agents as needed, specifically targeting open-source Monoscope through OTLP. Scope now includes necessary dependency updates/downloads, isolated package/bot implementation, verification and one independent reviewer with at most two fix/recheck rounds. No commits, pushes, releases, server changes, live telemetry or deployment authorized. Earlier Sentry release authority is not authority for this work.

Canonical cross-repository design lives here. Existing `docs/specs/sentry/spec.md` remains historical authority for Sentry behavior. On future approved implementation, bot-local specifications reference this document and record local scope/evidence; no competing design copies.

Exact dependency selection and isolated compatibility checks precede integration. Supervisor s7l-super-logging (w8E:p2); isolated implementers s7l-child-otlp (w8V:p1), s7l-child-otdoc (w8W:p1), s7l-child-otaudit (w8X:p1), s7l-child-otship (w8Y:p1). No recursive agents. Fresh independent reviewer starts only after author verification. Retain worktrees and evidence; no cleanup authority. Original auditing writers remain untouched.

## Intent

Each of three bots keeps JSON stdout/Dozzle and current Sentry Logs/explicit Issues, and optionally exports sanitized OTLP logs plus HTTP/job traces to Monoscope. Logging call sites continue using `observe.Logger`. Backend outages must not become business-operation failures.

Shared module means one root `go.mod`, one package version/release stream, and shared SDK dependency graph. It does not mean unused companion code is linked into every binary.

Non-goals: replace Sentry Issues/grouping; promise exception stacks/source maps/alerts/release parity; metrics; automatic instrumentation of every Google/DB/GOWA call; collector deployment; generic plugin API; custom queues/retry framework; remote configuration; durable telemetry delivery; application schema migrations; production rollout.

## Grounding and alternatives

Inspected baseline: pkg-logging `30a8db163a25c1fa223dfc9b0ba7cc205b945e96` (`v0.2.0`); document `fee818c25ad5db78690e1c7606d69156d90f285c`; auditing `75cd2ba8539ff9295c0d888cd9bcff34b0658a11`; shipping `da870adeabe7de036e56db78cab98cbbb8954caa`. Revalidate heads and dirty files before implementation; auditing has concurrent unrelated workflow/store edits. No checkout resets, bulk staging, or modifications to those edits.

Evidence: `observe/{logger,config,resource,redact,context,fields}.go`, `internal/output/output.go`, `observe/sentry/sentry.go`, gRPC interceptors, bot Sentry/logging/runtime/config files and release workflows.

- Existing destination fan-out already handles levels, resource/With binding, pre-output top-level redaction and record cloning. A second Logger wrapper would duplicate these invariants and private trace-generation policy: rejected.
- Shared-module `observe/otlp` companion reuses existing seam and Sentry pattern: selected by user. SDK requirements affect all consumers' module graphs, accepted trade-off.
- Nested module could isolate non-users' dependency graphs but adds separate tags/checks/version compatibility: rejected for current scope.
- Collector logs plus direct bot traces is viable but splits policy/ownership and reads richer local logs: not selected.
- Handwritten OTLP encoding/transport creates unnecessary protocol/lifecycle ownership: rejected.

Explicit narrow project-direction exception: optional companion may use official OTel logging bridge/API; bots own SDK/exporters/tracing. Core `observe` stays vendor-neutral and imports no OTel SDK. Existing no-exporter-framework/no-tracing-in-core guidance remains. Update AGENTS/INTEGRATION wording in implementation to state exception rather than silently contradict guidance.

## Ownership and proposed API

### Package

Add `observe/otlp`, package name `observeotlp`, with proposed constructor:

```go
func New(provider log.LoggerProvider) (observe.Destination, error)
```

Here `log` is `go.opentelemetry.io/otel/log`. This signature is a design target pending official bridge compatibility verification, not an existing API. Reject nil provider; never silently select global provider. Typed-nil behavior must be defined/tested before finalizing API.

Use official `go.opentelemetry.io/contrib/bridges/otelslog` handler with explicitly supplied provider, then existing internal destination constructor. Do not expose slog in public signatures. Do not add new public core fields/methods or a public arbitrary-handler constructor. Do not add a second Go module.

Companion owns log conversion only. It neither creates exporters/resources nor flushes/shuts down providers. Official bridge capability, attribute binding, context correlation and timestamp/severity behavior must be inspected before coding. If bridge cannot meet acceptance, return for design decision rather than introduce a custom framework.

### Bots

Each bot owns explicit logger/tracer providers, HTTP/protobuf exporters, trusted Resource, privacy processors, sampling, instrumentation, configuration and shutdown. No `otel.Set*Provider`, global propagator changes or global Sentry changes. Pass providers/propagators explicitly.

Compose stdout plus Sentry plus OTLP destinations with existing `observe.New`. Keep Sentry client configuration/capture behavior unchanged. Default common minimum remains `Config.Level`; OTLP privacy policy may further discard events. Failure diagnostics use a separate local-only logger or stderr, never same remote fan-out.

### Server

Supplied verified handoff, not revalidated in this task: Monoscope at `dito-asus-laptop`, Tailscale `100.81.4.38`, stack `/home/dito/stacks/monoscope`, image `ghcr.io/monoscope-tech/monoscope@sha256:a1a3daadd49e8da5e9cc5ce3825f9530b55c267bf597c54a28610c66aaa89b9a`. OTLP HTTP `/v1/logs` and `/v1/traces` on port 18080 accept `application/x-protobuf`, project `x-api-key`. UI Basic Auth is separate; DB and gRPC are not published. No server changes needed by design. No credentials in spec. Demo proves ingestion only; four-hour soak result unknown.

## Processing and privacy invariants

1. Existing logger sanitizes top-level fields, binds trusted resource and With fields, then performs current fan-out. No change to stdout JSON schema or Sentry routing.
2. OTLP bridge constructs log record with native context correlation when valid active span context exists.
3. Bot-owned privacy processor runs before batch processor/exporter, rebuilding body and attributes from approved event/value vocabulary. Missing/unknown event must stop downstream processing, not merely clear record body. Verify SDK short-circuit semantics; compose privacy wrapper around downstream processor if SDK otherwise continues after rejection. Exactly one approved processing path reaches exporter.
4. Batch SDK owns bounded enqueue/export/retry. No network export synchronously in logging call; full queue drops instead of blocking business work.
5. Trusted resource and instrumentation scope are reviewed separately: log attribute filtering cannot sanitize provider Resource. No automatic host/user/process/environment detectors or arbitrary SDK enrichment.

Current core masking does not sanitize messages, raw errors or nested objects. Neither remote path may rely on it alone. Resolve duplicate attributes deterministically before policy decisions: last effective value across resource/With/call ordering; reject wrong types. Strip all unknown attributes/groups/objects. Bound attribute counts, body lengths and approved numeric values. No raw errors, bodies, phone/chat/message/dedup IDs, headers, credential paths, full URLs or arbitrary baggage in exports.

Existing bot policies are baseline, not permission to broaden remote data:

| Bot | Preserve for Monoscope logs |
| --- | --- |
| Document | Closed `remoteEvents` vocabulary; fixed event-name body; drop ordinary attributes. Existing per-log Sentry trace identity stays unchanged. |
| Auditing | Closed events; fixed event-name body; finite numbers in `0..1e9` for `http.status_code`, `job.attempt`, `duration_ms`, `queue.budget`, `queue.jobs`; closed `outcome`/`job.transition` values. |
| Shipping | Six fixed-message events; closed error-code, operation and stage values; no arbitrary identifiers. |

Where shared policy extraction is useful, extract code-owned safe-event decisions, not Sentry/OTLP objects. Preserve exact Sentry acceptance/rejection/output semantics with regressions. Do not broaden remote event coverage merely to demonstrate traces.

Only new remote metadata approved by this design: trusted service/version/environment and SDK-native trace/span correlation. Native IDs must not encode business identifiers. Existing arbitrary `request.id`, `correlation.id`, `trace.id` remain excluded from remote policy. No copying SDK enrichment from Sentry.

## Identity and configuration

OTel Resource: trusted `service.name`, nonempty `service.version` when known, and `deployment.environment.name`. Preserve existing dotted stdout keys including `env`; do not rename local schema. Document/auditing reuse existing service/version/environment config; shipping uses existing build-injected `versioninfo.Version` and explicit environment default consistent with current runtime behavior. Do not put project key in Resource.

Use standard OTLP variables: `OTEL_EXPORTER_OTLP_ENDPOINT` or signal-specific `OTEL_EXPORTER_OTLP_LOGS_ENDPOINT` / `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`, protocol `http/protobuf`, and headers via base or signal-specific `OTEL_EXPORTER_OTLP_*HEADERS`. Honor standard signal-specific precedence and base-path construction; document and test exact paths. Never append `/v1/logs` twice. No Sentry DSN translation or reuse.

Target examples without credentials:

- `http://100.81.4.38:18080/v1/logs`
- `http://100.81.4.38:18080/v1/traces`

Both absent: Monoscope disabled, existing stdout/Sentry unaffected. Validate both signal setups before attaching destination/instrumentation; partial/malformed enabled configuration disables Monoscope as a unit with fixed local-only warning. Never print raw SDK URL/header/error contents. Runtime failure of one Monoscope signal does not disable other signal or Sentry. Existing invalid-Sentry behavior is out of scope.

Validate scheme/host/path, reject URL credentials/query/fragment, constrain headers to approved project-auth contract, disallow cross-origin credential redirects, and never disable TLS verification. HTTP is permitted only for explicitly approved private target; no public plaintext endpoint fallback. Endpoint/key changes require operational approval. Verify bot hosts can reach Tailscale address during later separately authorized rollout.

## Native traces and bot touch points

Use official SDK TraceContext propagation and explicit tracer provider. Preserve local legacy metadata independently; never interpret arbitrary `x-trace-id` as native parent. Accept only validated W3C trace context under documented ingress trust policy; no incoming baggage propagation. Trace state is discarded unless separately reviewed.

Initial trace coverage: webhook server handling and meaningful scheduled/job execution. Span names use fixed route templates/operation names; never IDs/raw paths/query values. Health checks and idle worker polling are excluded. Initial synthetic validation samples all selected spans; production sampling/volume setting must be explicit before rollout. Traces do not create additional completion logs.

Spans carry allowlisted route/status/operation/error-code attributes only; no automatic request headers, URL queries, DB statements, payloads or raw exception recording. Do not call `RecordError` with unsanitized business errors. Mark unexpected/system failures as Error, preserve expected business/cancellation semantics without creating synthetic Sentry Issues. Root/system startup failures may produce logs/Issues without inventing request spans.

Request child spans propagate context through function calls. Detached in-process work retains safe span context but gets independent cancellation/deadline; do not reuse mutable scope. Durable work stores only validated trace context in dedicated optional queue metadata, never baggage/business identifiers as trace IDs. Queue persistence changes require compatibility review before implementation; if safe additive metadata is not supported, start independent execution root spans and explicitly document loss of enqueue/execution linkage. No irreversible queue migration authorized by this spec.

| Repository | Planned areas |
| --- | --- |
| pkg-logging | `observe/otlp/*`, tests; root go.mod/go.sum; AGENTS, README, INTEGRATION, CHANGELOG. Existing output/core change only if verified necessity returns to approval. |
| document | `internal/observability` privacy/provider/context helpers; `config`; `cmd/bot/main.go`; webhook/reminder/background boundaries; tests/docs. Current traceparent substring parser must not supply native SDK parent context. |
| auditing | `cmd/bot/adapter` provider/logger/webhook/queue/reminder boundaries; config/tests/docs. Coordinate with concurrent writers before overlapping edits; preserve workflow/store work. |
| shipping | `cmd/bot` provider/logger/request/worker lifecycle; env example/tests/docs; reuse versioninfo. HTTP handler wrappers must preserve ResponseWriter capabilities. |

Shipping refinement approved during implementation: one optional consumer-owned `Service.StartJob func(context.Context, string) (context.Context, func(error))` hook in app/shipping, invoked only after successful claims/export due check. Concrete tracer stays in cmd/bot; nil hook preserves behavior. Tests must prove no idle spans and completion on each actual-job return. No OTel imports in app/domain. No OTel SDK imports in pure domains. No unrelated architecture refactors. Automatic external HTTP instrumentation is deferred until URL/privacy policy is proven; existing indirect otelhttp dependency is not proof of safe enabled instrumentation.

## Failure, concurrency and shutdown

Proposed starting bounds (verify supported options against selected official versions): each signal queue 512 entries, batch at most 128, scheduled export every 1 second, export timeout 1 second and total per-batch retry budget at most 1 second. No unbounded queues, infinite retries, or per-record flush. Configure finite transport dial/header/body behavior. Queue pressure drops records, reports aggregated local-only diagnostics, and never blocks handlers waiting for capacity. SDK batch-count bounds must be combined with payload/attribute limits; not claimed as exact byte bound.

Privacy processors/With derivation must be race-safe, immutable per record, and not mutate sibling destination records. No shared request state in providers. Async network isolation does not change existing synchronous JSON/Sentry handler blocking limits; this work must not claim to fix arbitrary blocked stdout writers.

Shutdown order: stop intake; cancel workers; bounded request/worker drain (existing 10-second phase); end spans; freeze further telemetry emission; run both OTel provider shutdowns and Sentry flush/close within shared 2-second telemetry budget using fresh non-canceled context. Do not give each exporter a new serial 2-second budget. Report unsuccessful flush locally; close resources without an unbounded second drain. No repeated provider shutdown or late worker use-after-close.

Shipping currently calls unbounded `wg.Wait()` after HTTP shutdown. Add bounded waiting and verify cancellation behavior at telemetry lifecycle seam, without changing business semantics. Document/auditing can also outlive drain deadline: freeze/drop subsequent emissions safely before shutdown rather than promise all work drained. Best-effort telemetry may be lost on deadline, crash, SIGKILL, outage or queue pressure. Telemetry health must not control bot readiness or workflow success.

## Acceptance and verification

1. Existing core/Sentry API, stdout schema, minimum levels, masking, trace-generation policy and one gRPC completion log per invocation remain unchanged. Existing package/bot regressions pass.
2. OTLP destination handles nil validation, resource/With/call attributes, duplicate resolution, parent/sibling isolation and concurrent calls. No slog in public API; no provider globals.
3. Fake in-memory exporter and local HTTP/protobuf receiver observe sanitized records only. Unknown events are absent, not empty exported records. Wrong types, duplicate-key spoofing, nested values and numeric boundary/nonfinite values exercise fail-closed policy.
4. Secret sentinels across bodies/errors/nested attrs/With/Resource/SDK enrichment/incoming headers/baggage are absent from decoded OTLP. Auth header sent only to approved receiver, never logs; redirect tests prevent credential leakage.
5. Sentry fake transport golden assertions preserve current Logs/Issue allowlists, fixed messages, fingerprints, cancellation exclusions and client-scope isolation. One logging call is not an automatic Issue.
6. Selected synthetic HTTP request and job spans have correct timing, resource identity, validated parent/child relation and log TraceId/SpanId. Invalid/zero/injected legacy IDs cannot create native parents. Concurrent jobs have no context bleed. Idle polls/health checks excluded.
7. Missing/invalid/partial OTLP config exercises disable/fixed-warning behavior, endpoint precedence, protobuf protocol/path correctness and secret-free diagnostics. Sentry remains independent.
8. Receiver cases: offline, slow/stalled body, 401/403, 429, 5xx, malformed/partial-success response and queue saturation. Memory/queue/retry are finite; no telemetry-induced business failure; accepted/dropped counts reconcile within documented best-effort limits. Partial rejection is not treated as full delivery.
9. Shutdown with active jobs/unavailable receiver stays within drain plus telemetry budget (scheduler tolerance documented), emits no recursive diagnostics, and safely handles late logs. Shipping no longer waits forever at wg.Wait.
10. Main-module import graph for core contains no OTel SDK import; main module graph intentionally includes companion dependencies. Root tests cover companion; no nested module or workspace replacement required. Published-module-style checks use GOWORK=off.
11. Required later checks: native `go test ./...`, `go test -race ./...`, `go vet ./...`; each bot build; package Go 1.25 compatibility and bot declared Go toolchains. Repository `rtk go test ./...` preferred when installed. No framework added for tests.
12. Later separately authorized synthetic end-to-end verifies stored/UI logs and traces plus continued Sentry Issue behavior using fake bot dependencies, not real WhatsApp/Sheets activity. Do not claim backend feature parity or soak success without evidence.

## Delivery, approval gates and rollback

1. Dependency/API gate: inspect official otelslog/log/SDK/exporter source/docs; propose exact aligned pins and check release metadata, Go requirements, provider/processor rejection semantics, export retry bounds and source compatibility. Cached modules are not proof of safe versions. No dependency changes authorized yet. Raise Go-baseline change for approval rather than silently upgrading.
2. Approve spec, pins and production sampling/ingress trust policy as needed. Authorize implementation scope explicitly. Coordinate auditing ownership before edits.
3. Implement companion plus fake exporter/transport regression tests; verify package and Sentry behavior first.
4. Implement one bot vertical slice, then reuse proven pattern in remaining bots with per-bot policies; verify scope against dirty baseline before each edit. No live workflows.
5. Complete checks and separately authorized independent review of security/intent/sufficiency/simplicity/project direction; fix/recheck findings. Review authority not inherited from earlier release.
6. Only after separate approval: release package, pin bots, release bots, configure keys/endpoints, deploy and validate. No versions selected or rollout authorized now; do not touch shipping auto-deploy variable during specification work.

Rollback: unset OTLP endpoint configuration to return to existing stdout/Sentry behavior. No storage migration or deletion required. Sentry DSN remains independent. If code rollback needed, revert scoped telemetry changes without touching concurrent work. Queue metadata, if later approved, must tolerate old readers/writers before rollout.

## Open gates and evidence limits

- Shared module, both-backend direction and implementation now approved. Explicit roots/no external ingress parents and sample-all synthetic checks are implementation defaults; production rollout remains separately gated.
- Initial inspected compatible pins: otelslog v0.20.0; OTel stable API/SDK/trace exporter v1.45.0; log API/SDK/log exporter v0.21.0. Cached/downloaded official module metadata declares Go 1.25.0. Final bridge/processor/runtime compatibility still requires author checks and isolated verification.
- Production sampling and acceptance of external W3C parents require explicit policy before rollout; locally generated roots sufficient for initial fake tests.
- Durable enqueue/execution linkage depends on safe compatible queue metadata; independent execution roots are bounded fallback, not claimed end-to-end trace.
- Implementation and bounded agent/reviewer work authorized. Package review accepted; bot review pending. No remote checks run. Acceptance evidence follows below.

## Implementation evidence

Package source implemented in isolated s7l-child-otlp, integrated into original pkg-logging checkout after review. Public constructor matches proposed API and rejects nil/typed-nil providers. Existing core/Sentry source unchanged. Exact package pins: otelslog v0.20.0; OTel log API/SDK v0.21.0; stable OTel v1.45.0; package retains gRPC v1.80.0 and Go1.25.0. Bot HTTP exporters require gRPC v1.83.0, whose gax-go/v2 v2.17.0 requirement raises Google API to v0.264.0; this necessary transitive upgrade requires bot store regression coverage.

Author reports GOWORK=off GOTOOLCHAIN=go1.25.0 tests/race/vet/mod verify pass. Supervisor bounded Docker Go1.27.3 unit/race/vet passed on first package snapshot: network none, readonly source/modules/root, uid65534, empty allowlisted environment, no credentials/socket, CPU2, memory/swap3GiB, pids128, tmpfs3GiB, CPU300s/wall360s, nofile1024, file256MiB. Latest additional regression tests require final rerun.

Independent source reviewer s7l-child-otreview w80:p1, baseline 30a8db1, accepted package snapshot e90374cb48b83b2b605e1990480eb750de2d178c8587f8804ad24beb29ba911d. Nonblocking combined-three-output/disabled-provider test gaps fixed in observe/otlp/otlp_test.go; same reviewer accepted updated snapshot b2107d51ebe8dbd46a2bd5af83def37bf3bfb2938df897e894e707e58cc56920. Snapshot hashes use sorted relative path, NUL, file bytes. Reviewer read-only/source-only; no runtime security verification claimed. Bot review round1 found document invalid-config nil logger panic and shipping missing actual-worker native correlation / notification send error status. Authors fixed; r2 review accepted document guard but found startup regression could exit early without exercising branch. Shipping correlation fix exposed canceled command-context suppressing Sentry Issues and stale prior-job span attached to later claim errors. Final round2 fix/recheck requested; unresolved blockers after that stop per lifecycle. Do not claim acceptance from passing tests.

Supervisor network-none bounded Docker unit/race/vet/build passed initial and r2 snapshots for all three bots using GOWORK=off and local package replacement only in disposable tidy-resolved snapshots. Exact r2 source hashes: document 4193e982c8f98e74b8527ac7c3b15118cef4537840eff7e852082280a154aa9f; auditing 49701f024d0c7d73e6cb96fef31b59d59a9a05334e5b982c1bf59da51cdb6fbe; shipping 9f50aac2967408b0cfc33b73d5c5ca654414908824d2ac667ebe3421e7d3fa49. Bot runner bounds: CPU2, memory/swap3GiB, pids128, tmpfs3GiB, CPU500s/wall540s, nofile1024, file256MiB, readonly source/toolchain/module cache, empty allowlisted env, no credentials/socket, uid65534, caps dropped/no-new-privileges. Synthetic receiver traffic stays container loopback. This verifies local integration, not published dependency resolution. Final fixes completed within two bot fix/recheck rounds. Fresh reviewer accepted all four final source/test snapshots with no unresolved bounded blockers: package b2107d51ebe8dbd46a2bd5af83def37bf3bfb2938df897e894e707e58cc56920; document b7f01019bd86234df0e61c26076a315f214e3a380d141a56559405b42e315d11; auditing effdac1e73f4c548669cd556ad80a5252798b061a9fb41b8f68f78e2e974fb2a; shipping 8ac04d2102dbf29233ef2517bc89ca66f425d14b4c655e01cbeeb9c006b6b315. Reviewer verified hashes/diffs and source only; no target execution.

All exact final snapshots then passed supervisor isolated Docker unit/race/vet, plus bot builds, with bounds above. Logs retained under ignored `.scratch/otlp/*-final-check.log`. Additional coverage includes actual privacy-to-decoded-protobuf sentinels, wrong-type/unknown drops, both-signal pressure, bounded response matrices, shared fresh shutdown budget, active span ending during frozen teardown, late Sentry Logs/Issues controls and real shipping processBatch correlation/count regressions. Document startup regression deterministically reaches repaired branch; author scratch-only mutation without nil guard panicked. Inferred pressure drops are test accounting, not a new production delivery guarantee.

Reviewed scoped files integrated into original checkouts after verifying each changed path still matched child Git baseline and reviewed snapshot. Auditing's unrelated workflow/store changes and document attachment preserved. Ignored bot `.scratch/otlp/go.work` files reference original bot plus original pkg-logging checkout; use explicit GOWORK path for local tests. Tracked bot pins remain published v0.2.0, which lacks companion; no tracked local replacements or unreleased fabricated version added. Final scratch-only tidy copies supply dependency checksums for verification, not a release pin.

Local implementation accepted; published-consumer resolution remains a release blocker until separately authorized package release and bot pin updates. Production sampling/volume, endpoint/key approval/reachability, deployment, Monoscope stored/UI validation and soak remain open operational gates. No commits, pushes, releases, live telemetry, server changes or deployment performed.
