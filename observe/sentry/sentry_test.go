package observesentry_test

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
	observegrpc "github.com/deccara-tech/pkg-logging/observe/grpc"
	observesentry "github.com/deccara-tech/pkg-logging/observe/sentry"
	"github.com/getsentry/sentry-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type transport struct {
	mu     sync.Mutex
	events []*sentry.Event
}

func (*transport) Configure(sentry.ClientOptions)        {}
func (*transport) Flush(time.Duration) bool              { return true }
func (*transport) FlushWithContext(context.Context) bool { return true }
func (*transport) Close()                                {}
func (t *transport) SendEvent(e *sentry.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, e)
}
func (t *transport) logs() []sentry.Log {
	t.mu.Lock()
	defer t.mu.Unlock()
	var logs []sentry.Log
	for _, e := range t.events {
		logs = append(logs, e.Logs...)
	}
	return logs
}
func client(t *testing.T) (*sentry.Client, *transport) {
	t.Helper()
	tr := &transport{}
	c, err := sentry.NewClient(sentry.ClientOptions{Dsn: "https://public@example.invalid/1", Transport: tr})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	return c, tr
}
func cfg(w io.Writer) observe.Config {
	return observe.Config{Resource: observe.Resource{ServiceName: "audit-bot", Environment: "test"}, Writer: w, GenerateTraceID: observe.BoolPtr(false)}
}
func flush(t *testing.T, c *sentry.Client) {
	t.Helper()
	if !c.Flush(5 * time.Second) {
		t.Fatal("flush failed")
	}
}
func value(l sentry.Log, k string) string { return fmt.Sprint(l.Attributes[k].AsInterface()) }

func TestStructuredLogsPreserveFieldsRedactionAndContext(t *testing.T) {
	c, tr := client(t)
	destination, err := observesentry.New(c)
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	config := cfg(&stdout)
	config.Destinations = []observe.Destination{destination}
	config.Resource.ServiceVersion = "v1"
	config.RedactKeys = []string{"token", "service.version"}
	logger, err := observe.New(config)
	if err != nil {
		t.Fatal(err)
	}
	bound := logger.With(observe.String("correlation.id", "req-42"), observe.String("auth.token", "bound-secret"))
	child := bound.With(observe.String("job.id", "42"))
	scope := sentry.NewScope()
	scope.SetUser(sentry.User{ID: "synthetic-user-42"})
	ctx := sentry.SetHubOnContext(context.Background(), sentry.NewHub(c, scope))
	child.InfoContext(ctx, "job failed", observe.Event("job.failed"), observe.SensitiveString("private", "call-secret"))
	bound.InfoContext(context.Background(), "parent")
	logger.DebugContext(ctx, "filtered")
	if observe.GenerateTraceIDEnabled(child) {
		t.Fatal("disabled trace policy lost")
	}
	flush(t, c)
	logs := tr.logs()
	if len(logs) != 2 {
		t.Fatalf("logs=%d", len(logs))
	}
	first := logs[0]
	if got := value(first, "user.id"); got != "synthetic-user-42" {
		t.Fatalf("request scope lost: %q", got)
	}
	if _, ok := logs[1].Attributes["user.id"]; ok {
		t.Fatal("request scope leaked into background log")
	}
	for key, want := range map[string]string{"service.name": "audit-bot", "service.version": "[REDACTED]", "env": "test", "correlation.id": "req-42", "auth.token": "[REDACTED]", "job.id": "42", "event": "job.failed", "private": "[REDACTED]"} {
		if got := value(first, key); got != want {
			t.Errorf("%s=%q want %q", key, got, want)
		}
	}
	if _, ok := logs[1].Attributes["job.id"]; ok {
		t.Fatal("child fields leaked")
	}
	if first.Body != "job failed" || first.Level != sentry.LogLevelInfo {
		t.Fatalf("unexpected log %+v", first)
	}
	if bytes.Contains(stdout.Bytes(), []byte("secret")) {
		t.Fatal("secret in stdout")
	}
	dec := json.NewDecoder(&stdout)
	count := 0
	for dec.More() {
		var entry map[string]any
		if err := dec.Decode(&entry); err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count != 2 {
		t.Fatalf("stdout count=%d", count)
	}
	tr.mu.Lock()
	defer tr.mu.Unlock()
	for _, event := range tr.events {
		if len(event.Exception) > 0 {
			t.Fatal("log created exception")
		}
	}
}

func TestDestinationRoutesToSuppliedClientConcurrently(t *testing.T) {
	c, tr := client(t)
	other, otherTr := client(t)
	d, err := observesentry.New(c)
	if err != nil {
		t.Fatal(err)
	}
	config := cfg(io.Discard)
	config.Destinations = []observe.Destination{d}
	logger, err := observe.New(config)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sentry.SetHubOnContext(context.Background(), sentry.NewHub(other, sentry.NewScope()))
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) { defer wg.Done(); logger.With(observe.Int("job.id", i)).InfoContext(ctx, "job") }(i)
	}
	wg.Wait()
	flush(t, c)
	flush(t, other)
	if len(tr.logs()) != 20 || len(otherTr.logs()) != 0 {
		t.Fatal("wrong client or log count")
	}
}

func TestInvalidDestinations(t *testing.T) {
	if _, err := observesentry.New(nil); err == nil {
		t.Fatal("nil client accepted")
	}
	config := cfg(io.Discard)
	config.Destinations = []observe.Destination{{}}
	if _, err := observe.New(config); err == nil {
		t.Fatal("zero destination accepted")
	}
}

func TestClientCompletionOncePerDestination(t *testing.T) {
	c, tr := client(t)
	d, err := observesentry.New(c)
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	config := cfg(&stdout)
	config.Destinations = []observe.Destination{d}
	logger, err := observe.New(config)
	if err != nil {
		t.Fatal(err)
	}
	interceptor := observegrpc.UnaryClientInterceptor(logger)
	err = interceptor(context.Background(), "/audit.Service/Run", nil, nil, nil, func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		md, _ := metadata.FromOutgoingContext(ctx)
		if len(md.Get("x-trace-id")) != 0 {
			t.Fatal("trace generated despite disabled policy")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	flush(t, c)
	logs := tr.logs()
	if len(logs) != 1 || value(logs[0], "event") != "rpc.client.completed" {
		t.Fatalf("unexpected logs %+v", logs)
	}
	if bytes.Count(stdout.Bytes(), []byte("rpc.client.completed")) != 1 {
		t.Fatal("duplicate completion")
	}
}

func TestUnaryCompletionOncePerDestination(t *testing.T) {
	c, tr := client(t)
	d, err := observesentry.New(c)
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	config := cfg(&stdout)
	config.Destinations = []observe.Destination{d}
	logger, err := observe.New(config)
	if err != nil {
		t.Fatal(err)
	}
	interceptor := observegrpc.UnaryServerInterceptor(logger)
	_, err = interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/audit.Service/Run"}, func(ctx context.Context, _ any) (any, error) {
		if observe.MetadataFromContext(ctx).TraceID != "" {
			t.Fatal("trace generated despite disabled policy")
		}
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	flush(t, c)
	logs := tr.logs()
	if len(logs) != 1 || value(logs[0], "event") != "rpc.server.completed" {
		t.Fatalf("unexpected logs %+v", logs)
	}
	if bytes.Count(stdout.Bytes(), []byte("rpc.server.completed")) != 1 {
		t.Fatal("duplicate completion")
	}
}
