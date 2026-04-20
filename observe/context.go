package observe

import "context"

type loggerContextKey struct{}
type metadataContextKey struct{}

// RequestMetadata holds request-scoped correlation identifiers.
type RequestMetadata struct {
	RequestID     string
	CorrelationID string
	TraceID       string
	SourceService string
}

func (m RequestMetadata) fields() []Field {
	fields := make([]Field, 0, 4)
	if m.RequestID != "" {
		fields = append(fields, String("request.id", m.RequestID))
	}
	if m.CorrelationID != "" {
		fields = append(fields, String("correlation.id", m.CorrelationID))
	}
	if m.TraceID != "" {
		fields = append(fields, String("trace.id", m.TraceID))
	}
	if m.SourceService != "" {
		fields = append(fields, String("source.service", m.SourceService))
	}
	return fields
}

func IntoContext(ctx context.Context, logger Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if logger == nil {
		return ctx
	}
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

func FromContext(ctx context.Context) Logger {
	if ctx != nil {
		if logger, ok := ctx.Value(loggerContextKey{}).(Logger); ok && logger != nil {
			return logger
		}
	}
	return fallbackLogger()
}

func MetadataIntoContext(ctx context.Context, md RequestMetadata) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, metadataContextKey{}, md)
}

func MetadataFromContext(ctx context.Context) RequestMetadata {
	if ctx == nil {
		return RequestMetadata{}
	}
	md, _ := ctx.Value(metadataContextKey{}).(RequestMetadata)
	return md
}
