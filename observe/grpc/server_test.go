package observegrpc

import (
	"context"
	"net"
	"testing"

	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"github.com/deccara-tech/pkg-logging/observe"
	"github.com/deccara-tech/pkg-logging/observe/testkit"
)

func newGRPCTestLogger(t *testing.T) (observe.Logger, *testkit.Sink) {
	t.Helper()
	sink := testkit.NewSink()
	logger, err := observe.New(observe.Config{
		Resource: observe.Resource{
			ServiceName:    "order-service",
			ServiceVersion: "1.2.3",
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

func TestUnaryServerInterceptorCompletionLogAndContextMetadata(t *testing.T) {
	baseLogger, sink := newGRPCTestLogger(t)
	interceptor := UnaryServerInterceptor(baseLogger)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		headerRequestID, "req-123",
		headerCorrelationID, "corr-123",
		headerTraceID, "trace-123",
		headerSourceService, "edge-gateway",
	))
	ctx = peer.NewContext(ctx, &peer.Peer{Addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080}})

	info := &gogrpc.UnaryServerInfo{FullMethod: "/order.v1.OrderService/ApproveOrder"}
	_, err := interceptor(ctx, "request", info, func(ctx context.Context, req any) (any, error) {
		meta := observe.MetadataFromContext(ctx)
		if meta.RequestID != "req-123" {
			t.Fatalf("request id missing in handler context: %q", meta.RequestID)
		}
		if meta.CorrelationID != "corr-123" {
			t.Fatalf("correlation id missing in handler context: %q", meta.CorrelationID)
		}
		if meta.TraceID != "trace-123" {
			t.Fatalf("trace id missing in handler context: %q", meta.TraceID)
		}
		observe.FromContext(ctx).InfoContext(ctx, "domain log", observe.Event("order.approved"))
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}

	entries := testkit.RequireEntries(t, sink, 2)
	completion, ok := testkit.FindEntryByEvent(entries, observe.EventRPCServerCompleted)
	if !ok {
		t.Fatalf("completion log not found: %#v", entries)
	}

	testkit.RequireFieldEqual(t, completion, "service.name", "order-service")
	testkit.RequireFieldEqual(t, completion, "service.version", "1.2.3")
	testkit.RequireFieldEqual(t, completion, "env", "test")
	testkit.RequireFieldEqual(t, completion, "rpc.system", "grpc")
	testkit.RequireFieldEqual(t, completion, "grpc.service", "order.v1.OrderService")
	testkit.RequireFieldEqual(t, completion, "grpc.method", "ApproveOrder")
	testkit.RequireFieldEqual(t, completion, "grpc.status_code", "OK")
	testkit.RequireFieldEqual(t, completion, "request.id", "req-123")
	testkit.RequireFieldEqual(t, completion, "correlation.id", "corr-123")
	testkit.RequireFieldEqual(t, completion, "trace.id", "trace-123")
	testkit.RequireFieldEqual(t, completion, "outcome", "success")
	testkit.RequireFieldNotEmpty(t, completion, "duration_ms")
	testkit.RequireFieldNotEmpty(t, completion, "peer.address")
}

func TestUnaryServerInterceptorHonorsDisabledTraceGeneration(t *testing.T) {
	sink := testkit.NewSink()
	baseLogger, err := observe.New(observe.Config{
		Resource: observe.Resource{
			ServiceName: "order-service",
			Environment: "test",
		},
		Writer:          sink,
		GenerateTraceID: observe.BoolPtr(false),
	})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	interceptor := UnaryServerInterceptor(baseLogger)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		headerRequestID, "req-abc",
		headerCorrelationID, "corr-abc",
	))
	info := &gogrpc.UnaryServerInfo{FullMethod: "/order.v1.OrderService/ApproveOrder"}

	_, err = interceptor(ctx, "request", info, func(ctx context.Context, req any) (any, error) {
		meta := observe.MetadataFromContext(ctx)
		if meta.TraceID != "" {
			t.Fatalf("trace id should not be generated in handler metadata: %q", meta.TraceID)
		}
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}

	entries := testkit.RequireEntries(t, sink, 1)
	completion, ok := testkit.FindEntryByEvent(entries, observe.EventRPCServerCompleted)
	if !ok {
		t.Fatalf("completion log not found: %#v", entries)
	}
	if _, hasTrace := completion["trace.id"]; hasTrace {
		t.Fatalf("trace.id field should be absent when generation disabled: %#v", completion)
	}
}
