package observegrpc

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"

	"github.com/deccara-tech/pkg-logging/observe"
)

func TestExtractIncomingMetadataPreservesValues(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		headerRequestID, "req-1",
		headerCorrelationID, "corr-1",
		headerTraceID, "trace-1",
		headerSourceService, "gateway",
	))

	meta := ExtractIncomingMetadata(ctx)
	if meta.RequestID != "req-1" {
		t.Fatalf("request id mismatch: %q", meta.RequestID)
	}
	if meta.CorrelationID != "corr-1" {
		t.Fatalf("correlation id mismatch: %q", meta.CorrelationID)
	}
	if meta.TraceID != "trace-1" {
		t.Fatalf("trace id mismatch: %q", meta.TraceID)
	}
	if meta.SourceService != "gateway" {
		t.Fatalf("source service mismatch: %q", meta.SourceService)
	}
}

func TestEnsureRequestMetadataFallbacks(t *testing.T) {
	meta := EnsureRequestMetadata(observe.RequestMetadata{})
	if meta.RequestID == "" {
		t.Fatal("request id must be generated")
	}
	if meta.CorrelationID == "" {
		t.Fatal("correlation id must be generated")
	}
	if meta.CorrelationID != meta.RequestID {
		t.Fatalf("correlation id should default to request id: %q vs %q", meta.CorrelationID, meta.RequestID)
	}
	if meta.TraceID == "" {
		t.Fatal("trace id must be generated")
	}
}

func TestEnsureRequestMetadataWithPolicy_DisableTraceGeneration(t *testing.T) {
	meta := EnsureRequestMetadataWithPolicy(observe.RequestMetadata{}, false)
	if meta.RequestID == "" {
		t.Fatal("request id must be generated")
	}
	if meta.CorrelationID == "" {
		t.Fatal("correlation id must be generated")
	}
	if meta.TraceID != "" {
		t.Fatalf("trace id should stay empty when generation disabled: %q", meta.TraceID)
	}
}

func TestInjectOutgoingMetadataDoesNotOverwriteExisting(t *testing.T) {
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		headerRequestID, "req-existing",
		headerCorrelationID, "corr-existing",
	))

	ctx = InjectOutgoingMetadata(ctx, observe.RequestMetadata{
		RequestID:     "req-new",
		CorrelationID: "corr-new",
		TraceID:       "trace-new",
		SourceService: "order-service",
	})

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("outgoing metadata missing")
	}
	if got := md.Get(headerRequestID); len(got) == 0 || got[0] != "req-existing" {
		t.Fatalf("request id overwritten: %v", got)
	}
	if got := md.Get(headerCorrelationID); len(got) == 0 || got[0] != "corr-existing" {
		t.Fatalf("correlation id overwritten: %v", got)
	}
	if got := md.Get(headerTraceID); len(got) == 0 || got[0] != "trace-new" {
		t.Fatalf("trace id mismatch: %v", got)
	}
	if got := md.Get(headerSourceService); len(got) == 0 || got[0] != "order-service" {
		t.Fatalf("source service mismatch: %v", got)
	}
}
