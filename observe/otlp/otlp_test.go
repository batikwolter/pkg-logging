package observeotlp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/deccara-tech/pkg-logging/observe"
	observeotlp "github.com/deccara-tech/pkg-logging/observe/otlp"
	observesentry "github.com/deccara-tech/pkg-logging/observe/sentry"
	"github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/trace"
)

// capture runs synchronously; no exporters, servers or SDK globals.
type capture struct {
	mu      sync.Mutex
	records []sdklog.Record
}

func (*capture) Enabled(context.Context, sdklog.EnabledParameters) bool { return true }
func (*capture) Shutdown(context.Context) error                         { return nil }
func (*capture) ForceFlush(context.Context) error                       { return nil }
func (c *capture) OnEmit(_ context.Context, r *sdklog.Record) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.records = append(c.records, r.Clone())
	return nil
}

// Embed the official no-op provider to satisfy OTel's forward-compatibility marker.
type pointerProvider struct{ noop.LoggerProvider }

func TestNewValidatesProvider(t *testing.T) {
	var pointer *pointerProvider
	var sdk *sdklog.LoggerProvider
	for _, provider := range []log.LoggerProvider{nil, pointer, sdk} {
		d, err := observeotlp.New(provider)
		if err == nil || d.Valid() {
			t.Fatalf("accepted nil provider %T", provider)
		}
	}
	for _, provider := range []log.LoggerProvider{noop.NewLoggerProvider(), &pointerProvider{}} {
		d, err := observeotlp.New(provider)
		if err != nil || !d.Valid() {
			t.Fatalf("rejected provider %T: %v", provider, err)
		}
	}
}

func newLogger(t *testing.T, writer io.Writer, level string) (observe.Logger, *capture) {
	t.Helper()
	c := &capture{}
	provider := sdklog.NewLoggerProvider(sdklog.WithProcessor(c))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := provider.Shutdown(ctx); err != nil {
			t.Error(err)
		}
	})
	d, err := observeotlp.New(provider)
	if err != nil {
		t.Fatal(err)
	}
	logger, err := observe.New(observe.Config{
		Resource: observe.Resource{ServiceName: "synthetic", ServiceVersion: "v1", Environment: "test"},
		Writer:   writer, Level: level, Destinations: []observe.Destination{d},
		GenerateTraceID: observe.BoolPtr(false),
	})
	if err != nil {
		t.Fatal(err)
	}
	return logger, c
}

func attrs(r sdklog.Record) map[string]attribute.Value {
	m := make(map[string]attribute.Value)
	r.WalkAttributes(func(kv attribute.KeyValue) bool {
		m[string(kv.Key)] = kv.Value
		return true
	})
	return m
}

func TestConversionBindingAndIsolation(t *testing.T) {
	var stdout bytes.Buffer
	logger, c := newLogger(t, &stdout, "debug")
	parent := logger.With(observe.String("bound", "parent"), observe.String("auth.token", "bound-secret"))
	child := parent.With(observe.String("child", "yes"), observe.String("bound", "child"))
	sibling := parent.With(observe.String("sibling", "yes"))
	before := time.Now()
	child.InfoContext(nil, "synthetic message", observe.Event("job.ready"),
		observe.String("bound", "call"), observe.String("password", "call-secret"),
		observe.Int("count", 7), observe.Bool("ok", true), observe.Float64("fraction", 0.5),
		observe.Duration("elapsed", 2*time.Millisecond))
	parent.WarnContext(context.Background(), "parent")
	sibling.ErrorContext(context.Background(), "sibling")
	logger.DebugContext(context.Background(), "root")
	if observe.GenerateTraceIDEnabled(child) {
		t.Fatal("trace-generation policy changed")
	}
	if len(c.records) != 4 {
		t.Fatalf("records=%d", len(c.records))
	}
	r := c.records[0]
	if r.Body().AsString() != "synthetic message" || r.Severity() != log.SeverityInfo || r.SeverityText() != "INFO" {
		t.Fatalf("body/severity: %v/%v/%s", r.Body(), r.Severity(), r.SeverityText())
	}
	if r.Timestamp().Before(before) || r.Timestamp().After(time.Now()) || r.ObservedTimestamp().IsZero() {
		t.Fatal("timestamp lost")
	}
	if got := r.InstrumentationScope().Name; got != "github.com/deccara-tech/pkg-logging/observe/otlp" {
		t.Fatalf("scope=%q", got)
	}
	m := attrs(r)
	for key, want := range map[string]string{
		"service.name": "synthetic", "service.version": "v1", "env": "test",
		"bound": "call", "child": "yes", "event": "job.ready",
		"auth.token": "[REDACTED]", "password": "[REDACTED]",
	} {
		if got := m[key].AsString(); got != want {
			t.Errorf("%s=%q want %q", key, got, want)
		}
	}
	if m["count"].AsInt64() != 7 || !m["ok"].AsBool() || m["fraction"].AsFloat64() != 0.5 || m["elapsed"].AsInt64() != int64(2*time.Millisecond) {
		t.Fatal("typed values changed")
	}
	for i, severity := range []log.Severity{log.SeverityInfo, log.SeverityWarn, log.SeverityError, log.SeverityDebug} {
		if c.records[i].Severity() != severity {
			t.Errorf("record %d severity=%v", i, c.records[i].Severity())
		}
	}
	for i := 1; i < 4; i++ {
		if _, ok := attrs(c.records[i])["child"]; ok {
			t.Fatalf("child field leaked into record %d", i)
		}
	}
	if attrs(c.records[1])["bound"].AsString() != "parent" || attrs(c.records[2])["bound"].AsString() != "parent" {
		t.Fatal("child mutated parent or sibling")
	}
	if _, ok := attrs(c.records[3])["bound"]; ok {
		t.Fatal("bound field leaked into root")
	}
	if bytes.Contains(stdout.Bytes(), []byte("bound-secret")) || bytes.Contains(stdout.Bytes(), []byte("call-secret")) {
		t.Fatal("stdout redaction regressed")
	}
	var local map[string]any
	if err := json.NewDecoder(&stdout).Decode(&local); err != nil {
		t.Fatal(err)
	}
	if local["msg"] != "synthetic message" || local["env"] != "test" {
		t.Fatal("stdout schema changed")
	}
}

func TestContextCorrelationAndMinimumLevel(t *testing.T) {
	logger, c := newLogger(t, io.Discard, "warn")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{1}, SpanID: trace.SpanID{2}, TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	logger.InfoContext(ctx, "filtered")
	logger.WarnContext(ctx, "correlated", observe.String("trace.id", "legacy-not-native"))
	logger.ErrorContext(context.Background(), "uncorrelated", observe.String("trace.id", sc.TraceID().String()))
	if len(c.records) != 2 {
		t.Fatalf("records=%d", len(c.records))
	}
	if r := c.records[0]; r.TraceID() != sc.TraceID() || r.SpanID() != sc.SpanID() || r.TraceFlags() != sc.TraceFlags() {
		t.Fatal("native context correlation lost")
	}
	if r := c.records[1]; r.TraceID().IsValid() || r.SpanID().IsValid() {
		t.Fatal("legacy attribute became native correlation")
	}
}

func TestExplicitProvidersRemainIsolated(t *testing.T) {
	first, a := newLogger(t, io.Discard, "info")
	second, b := newLogger(t, io.Discard, "info")
	first.InfoContext(context.Background(), "first")
	second.InfoContext(context.Background(), "second")
	first.InfoContext(context.Background(), "first again")
	if len(a.records) != 2 || len(b.records) != 1 {
		t.Fatalf("provider routing: first=%d second=%d", len(a.records), len(b.records))
	}
	if a.records[0].Body().AsString() != "first" || a.records[1].Body().AsString() != "first again" || b.records[0].Body().AsString() != "second" {
		t.Fatal("providers shared records")
	}
}

type sentryTransport struct {
	mu   sync.Mutex
	logs []sentry.Log
}

func (*sentryTransport) Configure(sentry.ClientOptions)        {}
func (*sentryTransport) Flush(time.Duration) bool              { return true }
func (*sentryTransport) FlushWithContext(context.Context) bool { return true }
func (*sentryTransport) Close()                                {}
func (tr *sentryTransport) SendEvent(event *sentry.Event) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.logs = append(tr.logs, event.Logs...)
}

func TestSiblingOutputsWithEnabledAndDisabledOTLP(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		t.Run(fmt.Sprintf("enabled=%t", enabled), func(t *testing.T) {
			c := &capture{}
			var provider log.LoggerProvider = noop.NewLoggerProvider()
			if enabled {
				p := sdklog.NewLoggerProvider(sdklog.WithProcessor(c))
				provider = p
				t.Cleanup(func() {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second)
					defer cancel()
					if err := p.Shutdown(ctx); err != nil {
						t.Error(err)
					}
				})
			}
			if got := provider.Logger("test").Enabled(context.Background(), log.EnabledParameters{Severity: log.SeverityInfo}); got != enabled {
				t.Fatalf("OTLP Enabled=%t want %t", got, enabled)
			}
			otlpDestination, err := observeotlp.New(provider)
			if err != nil {
				t.Fatal(err)
			}
			tr := &sentryTransport{}
			client, err := sentry.NewClient(sentry.ClientOptions{Dsn: "https://public@example.invalid/1", Transport: tr})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(client.Close)
			sentryDestination, err := observesentry.New(client)
			if err != nil {
				t.Fatal(err)
			}
			var stdout bytes.Buffer
			logger, err := observe.New(observe.Config{
				Resource: observe.Resource{ServiceName: "synthetic", Environment: "test"},
				Writer:   &stdout,
				// OTLP first ensures its disabled state cannot skip later siblings.
				Destinations: []observe.Destination{otlpDestination, sentryDestination},
			})
			if err != nil {
				t.Fatal(err)
			}
			logger.With(observe.String("bound", "shared")).InfoContext(context.Background(), "synthetic fanout", observe.Event("job.ready"))
			if !client.Flush(2 * time.Second) {
				t.Fatal("Sentry flush failed")
			}
			decoder := json.NewDecoder(&stdout)
			var local map[string]any
			if err := decoder.Decode(&local); err != nil {
				t.Fatal(err)
			}
			if local["msg"] != "synthetic fanout" || local["event"] != "job.ready" || local["bound"] != "shared" {
				t.Fatalf("stdout sibling changed: %v", local)
			}
			if err := decoder.Decode(&local); err != io.EOF {
				t.Fatalf("expected one stdout record, got %v", err)
			}
			tr.mu.Lock()
			defer tr.mu.Unlock()
			if len(tr.logs) != 1 {
				t.Fatalf("Sentry logs=%d", len(tr.logs))
			}
			remote := tr.logs[0]
			if remote.Body != "synthetic fanout" || remote.Attributes["event"].AsInterface() != "job.ready" || remote.Attributes["bound"].AsInterface() != "shared" {
				t.Fatalf("Sentry sibling changed: %v", remote)
			}
			want := 0
			if enabled {
				want = 1
			}
			if len(c.records) != want {
				t.Fatalf("OTLP records=%d want %d", len(c.records), want)
			}
			if enabled {
				r := c.records[0]
				if r.Body().AsString() != "synthetic fanout" || attrs(r)["event"].AsString() != "job.ready" || attrs(r)["bound"].AsString() != "shared" {
					t.Fatal("OTLP sibling changed")
				}
			}
		})
	}
}

func TestConcurrentDerivation(t *testing.T) {
	logger, c := newLogger(t, io.Discard, "info")
	parent := logger.With(observe.String("bound", "parent"))
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sc := trace.NewSpanContext(trace.SpanContextConfig{TraceID: trace.TraceID{byte(i + 1)}, SpanID: trace.SpanID{1}})
			ctx := trace.ContextWithSpanContext(context.Background(), sc)
			parent.With(observe.Int("worker", i)).InfoContext(ctx, fmt.Sprint(i))
		}(i)
	}
	wg.Wait()
	if len(c.records) != 32 {
		t.Fatalf("records=%d", len(c.records))
	}
	seen := map[int64]bool{}
	for _, r := range c.records {
		i := attrs(r)["worker"].AsInt64()
		if seen[i] || r.Body().AsString() != fmt.Sprint(i) || r.TraceID() != (trace.TraceID{byte(i + 1)}) || attrs(r)["bound"].AsString() != "parent" {
			t.Fatal("concurrent record state mixed")
		}
		seen[i] = true
	}
}
