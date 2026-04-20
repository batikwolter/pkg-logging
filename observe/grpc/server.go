package observegrpc

import (
	"context"
	"strings"
	"time"

	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/peer"

	"github.com/deccara-tech/pkg-logging/observe"
)

func UnaryServerInterceptor(baseLogger observe.Logger) gogrpc.UnaryServerInterceptor {
	if baseLogger == nil {
		baseLogger = observe.FromContext(context.Background())
	}

	return func(ctx context.Context, req any, info *gogrpc.UnaryServerInfo, handler gogrpc.UnaryHandler) (any, error) {
		start := time.Now()
		meta := ExtractIncomingMetadataWithPolicy(ctx, observe.GenerateTraceIDEnabled(baseLogger))
		ctx = observe.MetadataIntoContext(ctx, meta)

		requestLogger := baseLogger.With(metadataFields(meta)...)
		ctx = observe.IntoContext(ctx, requestLogger)

		resp, err := handler(ctx, req)

		classification := observe.ClassifyError(err)
		service, method := parseFullMethod(info.FullMethod)
		fields := []observe.Field{
			observe.Event(observe.EventRPCServerCompleted),
			observe.String("rpc.system", "grpc"),
			observe.String("grpc.service", service),
			observe.String("grpc.method", method),
			observe.String("grpc.status_code", classification.Code),
			observe.Float64("duration_ms", float64(time.Since(start).Microseconds())/1000),
			observe.String("outcome", classification.Outcome),
		}
		if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
			fields = append(fields, observe.String("peer.address", p.Addr.String()))
		}
		fields = append(fields, observe.ErrorFields(err)...)

		logByOutcome(requestLogger, ctx, err, "grpc server request completed", fields...)
		return resp, err
	}
}

func metadataFields(meta observe.RequestMetadata) []observe.Field {
	fields := make([]observe.Field, 0, 4)
	if meta.RequestID != "" {
		fields = append(fields, observe.String("request.id", meta.RequestID))
	}
	if meta.CorrelationID != "" {
		fields = append(fields, observe.String("correlation.id", meta.CorrelationID))
	}
	if meta.TraceID != "" {
		fields = append(fields, observe.String("trace.id", meta.TraceID))
	}
	if meta.SourceService != "" {
		fields = append(fields, observe.String("source.service", meta.SourceService))
	}
	return fields
}

func parseFullMethod(fullMethod string) (service string, method string) {
	trimmed := strings.TrimPrefix(fullMethod, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "unknown", trimmed
}

func logByOutcome(logger observe.Logger, ctx context.Context, err error, msg string, fields ...observe.Field) {
	if err == nil {
		logger.InfoContext(ctx, msg, fields...)
		return
	}
	if observe.IsBusinessError(err) {
		logger.WarnContext(ctx, msg, fields...)
		return
	}
	logger.ErrorContext(ctx, msg, fields...)
}
