package observe_test

import (
	"context"
	"testing"
	"time"

	"github.com/deccara-tech/pkg-logging/observe"
	"github.com/deccara-tech/pkg-logging/observe/testkit"
)

func newTestLogger(t *testing.T) (observe.Logger, *testkit.Sink) {
	t.Helper()
	sink := testkit.NewSink()
	logger, err := observe.New(observe.Config{
		Resource: observe.Resource{
			ServiceName:    "order-service",
			ServiceVersion: "1.0.0",
			Environment:    "test",
		},
		Writer: sink,
		Level:  observe.LevelDebug,
	})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	return logger, sink
}

func TestNewLoggerWritesStructuredJSON(t *testing.T) {
	logger, sink := newTestLogger(t)

	logger.InfoContext(context.Background(), "order approved",
		observe.Event("order.approved"),
		observe.String("order.id", "ord-1"),
		observe.Int("attempt", 2),
		observe.Bool("retry", false),
		observe.Float64("latency_ms", 12.5),
		observe.Duration("elapsed", 2*time.Second),
	)

	entries := testkit.RequireEntries(t, sink, 1)
	entry := entries[0]
	testkit.RequireFieldEqual(t, entry, "service.name", "order-service")
	testkit.RequireFieldEqual(t, entry, "service.version", "1.0.0")
	testkit.RequireFieldEqual(t, entry, "env", "test")
	testkit.RequireFieldEqual(t, entry, "event", "order.approved")
	testkit.RequireFieldEqual(t, entry, "order.id", "ord-1")
	testkit.RequireFieldEqual(t, entry, "attempt", "2")
}

func TestContextFallbackAndInjection(t *testing.T) {
	fallback := observe.FromContext(nil)
	fallback.InfoContext(nil, "fallback logger works", observe.Event("fallback.test"))

	logger, sink := newTestLogger(t)
	ctx := observe.IntoContext(context.Background(), logger)
	fromCtx := observe.FromContext(ctx)
	fromCtx.InfoContext(ctx, "context logger", observe.Event("context.test"))

	entries := testkit.RequireEntries(t, sink, 1)
	testkit.RequireFieldEqual(t, entries[0], "event", "context.test")
}

func TestRedaction(t *testing.T) {
	logger, sink := newTestLogger(t)
	logger.InfoContext(context.Background(), "login",
		observe.SensitiveString("password", "secret-value"),
		observe.String("auth.token", "abc123"),
	)

	entries := testkit.RequireEntries(t, sink, 1)
	entry := entries[0]
	testkit.RequireFieldEqual(t, entry, "password", "[REDACTED]")
	testkit.RequireFieldEqual(t, entry, "auth.token", "[REDACTED]")
}

func TestGenerateTraceIDConfig_DefaultTrue_AndDisable(t *testing.T) {
	defaultLogger, err := observe.New(observe.Config{
		Resource: observe.Resource{
			ServiceName: "order-service",
			Environment: "test",
		},
	})
	if err != nil {
		t.Fatalf("new default logger: %v", err)
	}
	if !observe.GenerateTraceIDEnabled(defaultLogger) {
		t.Fatal("default generateTraceID should be enabled")
	}

	disabledLogger, err := observe.New(observe.Config{
		Resource: observe.Resource{
			ServiceName: "order-service",
			Environment: "test",
		},
		GenerateTraceID: observe.BoolPtr(false),
	})
	if err != nil {
		t.Fatalf("new disabled logger: %v", err)
	}
	if observe.GenerateTraceIDEnabled(disabledLogger) {
		t.Fatal("generateTraceID should be disabled when configured false")
	}
}
