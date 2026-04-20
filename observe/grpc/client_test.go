package observegrpc

import (
	"context"
	"testing"

	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/deccara-tech/pkg-logging/observe"
	"github.com/deccara-tech/pkg-logging/observe/testkit"
)

func TestUnaryClientInterceptorPropagatesMetadataAndLogsCompletion(t *testing.T) {
	baseLogger, sink := newGRPCTestLogger(t)
	interceptor := UnaryClientInterceptor(baseLogger, WithSourceService("order-service"))

	ctx := observe.MetadataIntoContext(context.Background(), observe.RequestMetadata{
		RequestID:     "req-9",
		CorrelationID: "corr-9",
		TraceID:       "trace-9",
	})

	var outbound metadata.MD
	err := interceptor(ctx, "/billing.v1.BillingService/Charge", nil, nil, nil,
		func(ctx context.Context, method string, req, reply any, cc *gogrpc.ClientConn, opts ...gogrpc.CallOption) error {
			md, ok := metadata.FromOutgoingContext(ctx)
			if !ok {
				t.Fatal("missing outgoing metadata")
			}
			outbound = md
			return status.Error(codes.InvalidArgument, "bad request")
		})
	if err == nil {
		t.Fatal("expected invoker error")
	}

	if got := outbound.Get(headerRequestID); len(got) == 0 || got[0] != "req-9" {
		t.Fatalf("request id mismatch: %v", got)
	}
	if got := outbound.Get(headerCorrelationID); len(got) == 0 || got[0] != "corr-9" {
		t.Fatalf("correlation id mismatch: %v", got)
	}
	if got := outbound.Get(headerTraceID); len(got) == 0 || got[0] != "trace-9" {
		t.Fatalf("trace id mismatch: %v", got)
	}
	if got := outbound.Get(headerSourceService); len(got) == 0 || got[0] != "order-service" {
		t.Fatalf("source service mismatch: %v", got)
	}

	entries := testkit.RequireEntries(t, sink, 1)
	completion, ok := testkit.FindEntryByEvent(entries, observe.EventRPCClientCompleted)
	if !ok {
		t.Fatalf("client completion log not found: %#v", entries)
	}
	testkit.RequireFieldEqual(t, completion, "grpc.method", "Charge")
	testkit.RequireFieldEqual(t, completion, "grpc.status_code", "InvalidArgument")
	testkit.RequireFieldEqual(t, completion, "outcome", "error")
	testkit.RequireFieldEqual(t, completion, "request.id", "req-9")
	testkit.RequireFieldEqual(t, completion, "correlation.id", "corr-9")
	testkit.RequireFieldEqual(t, completion, "trace.id", "trace-9")
}

func TestUnaryClientInterceptorHonorsDisabledTraceGeneration(t *testing.T) {
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
	interceptor := UnaryClientInterceptor(baseLogger, WithSourceService("order-service"))

	ctx := observe.MetadataIntoContext(context.Background(), observe.RequestMetadata{
		RequestID:     "req-no-trace",
		CorrelationID: "corr-no-trace",
	})

	var outbound metadata.MD
	err = interceptor(ctx, "/billing.v1.BillingService/Charge", nil, nil, nil,
		func(ctx context.Context, method string, req, reply any, cc *gogrpc.ClientConn, opts ...gogrpc.CallOption) error {
			md, ok := metadata.FromOutgoingContext(ctx)
			if !ok {
				t.Fatal("missing outgoing metadata")
			}
			outbound = md
			return nil
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := outbound.Get(headerTraceID); len(got) != 0 {
		t.Fatalf("trace id should not be generated: %v", got)
	}

	entries := testkit.RequireEntries(t, sink, 1)
	completion, ok := testkit.FindEntryByEvent(entries, observe.EventRPCClientCompleted)
	if !ok {
		t.Fatalf("client completion log not found: %#v", entries)
	}
	if _, hasTrace := completion["trace.id"]; hasTrace {
		t.Fatalf("trace.id field should be absent when generation disabled: %#v", completion)
	}
}
