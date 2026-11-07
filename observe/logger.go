package observe

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/deccara-tech/pkg-logging/internal/output"
)

// Logger is the org-owned logging interface.
type Logger interface {
	DebugContext(ctx context.Context, msg string, fields ...Field)
	InfoContext(ctx context.Context, msg string, fields ...Field)
	WarnContext(ctx context.Context, msg string, fields ...Field)
	ErrorContext(ctx context.Context, msg string, fields ...Field)

	With(fields ...Field) Logger
}

type loggerImpl struct {
	backend         *slog.Logger
	redactor        redactor
	generateTraceID bool
}

func New(cfg Config) (Logger, error) {
	cfg = cfg.withDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	level := parseLevel(cfg.levelValue())
	var handler slog.Handler = slog.NewJSONHandler(cfg.writer(), &slog.HandlerOptions{
		AddSource: cfg.AddSource,
		Level:     level,
	})

	if len(cfg.Destinations) > 0 {
		handler = output.Compose(handler, level, cfg.Destinations)
	}
	base := slog.New(handler)
	impl := &loggerImpl{
		backend:         base,
		redactor:        newRedactor(cfg.RedactKeys),
		generateTraceID: cfg.shouldGenerateTraceID(),
	}
	if resourceFields := impl.redactor.sanitize(cfg.Resource.fields()); len(resourceFields) > 0 {
		impl.backend = impl.backend.With(fieldsToArgs(resourceFields)...)
	}

	return impl, nil
}

func (l *loggerImpl) DebugContext(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, slog.LevelDebug, msg, fields...)
}

func (l *loggerImpl) InfoContext(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, slog.LevelInfo, msg, fields...)
}

func (l *loggerImpl) WarnContext(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, slog.LevelWarn, msg, fields...)
}

func (l *loggerImpl) ErrorContext(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, slog.LevelError, msg, fields...)
}

func (l *loggerImpl) With(fields ...Field) Logger {
	safe := l.redactor.sanitize(fields)
	if len(safe) == 0 {
		return l
	}
	return &loggerImpl{
		backend:         l.backend.With(fieldsToArgs(safe)...),
		redactor:        l.redactor,
		generateTraceID: l.generateTraceID,
	}
}

func (l *loggerImpl) log(ctx context.Context, level slog.Level, msg string, fields ...Field) {
	if ctx == nil {
		ctx = context.Background()
	}
	safe := l.redactor.sanitize(fields)
	l.backend.Log(ctx, level, msg, fieldsToArgs(safe)...)
}

func fieldsToArgs(fields []Field) []any {
	args := make([]any, 0, len(fields)*2)
	for _, field := range fields {
		if strings.TrimSpace(field.Key) == "" {
			continue
		}
		args = append(args, field.Key, field.Value)
	}
	return args
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case LevelDebug:
		return slog.LevelDebug
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

var (
	fallbackOnce sync.Once
	fallbackLog  Logger
)

func fallbackLogger() Logger {
	fallbackOnce.Do(func() {
		logger, err := New(Config{
			Resource: Resource{
				ServiceName: "unknown-service",
				Environment: "unknown",
			},
		})
		if err != nil {
			fallbackLog = &loggerImpl{
				backend:         slog.Default(),
				redactor:        newRedactor(DefaultRedactKeys()),
				generateTraceID: true,
			}
			return
		}
		fallbackLog = logger
	})
	return fallbackLog
}

func (l *loggerImpl) traceIDGenerationEnabled() bool {
	return l.generateTraceID
}

func GenerateTraceIDEnabled(logger Logger) bool {
	type tracePolicy interface {
		traceIDGenerationEnabled() bool
	}
	p, ok := logger.(tracePolicy)
	if !ok {
		return true
	}
	return p.traceIDGenerationEnabled()
}
