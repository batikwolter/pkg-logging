// Package output connects in-module destinations without exposing backend types.
package output

import (
	"context"
	"errors"
	"log/slog"
)

// Destination is an opaque optional log destination. Its zero value is invalid.
// Obtain destinations from companion packages, not struct literals.
type Destination struct{ handler slog.Handler }

func New(handler slog.Handler) Destination { return Destination{handler: handler} }
func (d Destination) Valid() bool          { return d.handler != nil }

// Compose keeps the primary output first and applies a common minimum level.
func Compose(primary slog.Handler, level slog.Level, destinations []Destination) slog.Handler {
	handlers := []slog.Handler{primary}
	for _, d := range destinations {
		handlers = append(handlers, d.handler)
	}
	return &multi{handlers: handlers, level: level}
}

type multi struct {
	handlers []slog.Handler
	level    slog.Level
}

func (m *multi) Enabled(ctx context.Context, level slog.Level) bool {
	if level < m.level {
		return false
	}
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multi) Handle(ctx context.Context, r slog.Record) error {
	var errs []error
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			errs = append(errs, h.Handle(ctx, r.Clone()))
		}
	}
	return errors.Join(errs...)
}

func (m *multi) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithAttrs(append([]slog.Attr(nil), attrs...))
	}
	return &multi{handlers: handlers, level: m.level}
}

func (m *multi) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &multi{handlers: handlers, level: m.level}
}
