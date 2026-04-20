package observegrpc

import (
	"context"
	"strings"
	"time"

	gogrpc "google.golang.org/grpc"

	"github.com/deccara-tech/pkg-logging/observe"
)

type clientOptions struct {
	sourceService string
}

type ClientOption func(*clientOptions)

func WithSourceService(name string) ClientOption {
	return func(opts *clientOptions) {
		opts.sourceService = strings.TrimSpace(name)
	}
}

func UnaryClientInterceptor(baseLogger observe.Logger, options ...ClientOption) gogrpc.UnaryClientInterceptor {
	cfg := clientOptions{}
	for _, option := range options {
		option(&cfg)
	}

	return func(ctx context.Context, method string, req, reply any, cc *gogrpc.ClientConn, invoker gogrpc.UnaryInvoker, callOpts ...gogrpc.CallOption) error {
		logger := baseLogger
		if logger == nil {
			logger = observe.FromContext(ctx)
		}
		generateTraceID := observe.GenerateTraceIDEnabled(logger)

		meta := observe.MetadataFromContext(ctx)
		meta = EnsureRequestMetadataWithPolicy(meta, generateTraceID)
		if meta.SourceService == "" {
			meta.SourceService = cfg.sourceService
		}
		ctx = observe.MetadataIntoContext(ctx, meta)
		ctx = InjectOutgoingMetadataWithPolicy(ctx, meta, generateTraceID)
		logger = logger.With(metadataFields(meta)...)

		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, callOpts...)

		classification := observe.ClassifyError(err)
		_, parsedMethod := parseFullMethod(method)
		fields := []observe.Field{
			observe.Event(observe.EventRPCClientCompleted),
			observe.String("target.service", clientTarget(cc)),
			observe.String("grpc.method", parsedMethod),
			observe.String("grpc.status_code", classification.Code),
			observe.Float64("duration_ms", float64(time.Since(start).Microseconds())/1000),
			observe.String("outcome", classification.Outcome),
		}
		fields = append(fields, observe.ErrorFields(err)...)
		logByOutcome(logger, ctx, err, "grpc client request completed", fields...)
		return err
	}
}

func clientTarget(cc *gogrpc.ClientConn) string {
	if cc == nil {
		return "unknown"
	}
	target := strings.TrimSpace(cc.Target())
	if target == "" {
		return "unknown"
	}
	return target
}
