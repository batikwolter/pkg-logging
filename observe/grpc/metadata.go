package observegrpc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"

	"google.golang.org/grpc/metadata"

	"github.com/deccara-tech/pkg-logging/observe"
)

const (
	headerRequestID     = "x-request-id"
	headerCorrelationID = "x-correlation-id"
	headerTraceID       = "x-trace-id"
	headerSourceService = "x-source-service"
)

func ExtractIncomingMetadata(ctx context.Context) observe.RequestMetadata {
	return ExtractIncomingMetadataWithPolicy(ctx, true)
}

func ExtractIncomingMetadataWithPolicy(ctx context.Context, generateTraceID bool) observe.RequestMetadata {
	if ctx == nil {
		return EnsureRequestMetadataWithPolicy(observe.RequestMetadata{}, generateTraceID)
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return EnsureRequestMetadataWithPolicy(observe.RequestMetadata{}, generateTraceID)
	}
	meta := observe.RequestMetadata{
		RequestID:     first(md, headerRequestID),
		CorrelationID: first(md, headerCorrelationID),
		TraceID:       first(md, headerTraceID),
		SourceService: first(md, headerSourceService),
	}
	return EnsureRequestMetadataWithPolicy(meta, generateTraceID)
}

func EnsureRequestMetadata(meta observe.RequestMetadata) observe.RequestMetadata {
	return EnsureRequestMetadataWithPolicy(meta, true)
}

func EnsureRequestMetadataWithPolicy(meta observe.RequestMetadata, generateTraceID bool) observe.RequestMetadata {
	meta.RequestID = strings.TrimSpace(meta.RequestID)
	meta.CorrelationID = strings.TrimSpace(meta.CorrelationID)
	meta.TraceID = strings.TrimSpace(meta.TraceID)
	meta.SourceService = strings.TrimSpace(meta.SourceService)

	if meta.RequestID == "" {
		meta.RequestID = newID()
	}
	if meta.CorrelationID == "" {
		meta.CorrelationID = meta.RequestID
	}
	if generateTraceID && meta.TraceID == "" {
		meta.TraceID = newID()
	}
	return meta
}

func InjectOutgoingMetadata(ctx context.Context, meta observe.RequestMetadata) context.Context {
	return InjectOutgoingMetadataWithPolicy(ctx, meta, true)
}

func InjectOutgoingMetadataWithPolicy(ctx context.Context, meta observe.RequestMetadata, generateTraceID bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	meta = EnsureRequestMetadataWithPolicy(meta, generateTraceID)

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}

	setIfMissing(md, headerRequestID, meta.RequestID)
	setIfMissing(md, headerCorrelationID, meta.CorrelationID)
	setIfMissing(md, headerTraceID, meta.TraceID)
	if meta.SourceService != "" {
		setIfMissing(md, headerSourceService, meta.SourceService)
	}

	return metadata.NewOutgoingContext(ctx, md)
}

func first(md metadata.MD, key string) string {
	values := md.Get(strings.ToLower(key))
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func setIfMissing(md metadata.MD, key, value string) {
	if value == "" {
		return
	}
	if current := md.Get(strings.ToLower(key)); len(current) > 0 && strings.TrimSpace(current[0]) != "" {
		return
	}
	md.Set(key, value)
}

func newID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "generated-id"
	}
	return hex.EncodeToString(buf)
}
