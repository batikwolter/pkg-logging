// Package observesentry adds optional Sentry structured logs. Applications own
// SDK configuration, privacy callbacks, exception capture and shutdown flushing.
package observesentry

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/deccara-tech/pkg-logging/internal/output"
	"github.com/deccara-tech/pkg-logging/observe"
	"github.com/getsentry/sentry-go"
	sentryslog "github.com/getsentry/sentry-go/slog"
)

// New builds a destination for client without changing SDK globals. It emits
// structured logs, never exception events. Pass it in observe.Config.Destinations.
// Existing field redaction applies; messages and SDK scope need caller scrubbing.
func New(client *sentry.Client) (observe.Destination, error) {
	if client == nil {
		return observe.Destination{}, fmt.Errorf("sentry client is required")
	}
	ctx := sentry.SetHubOnContext(context.Background(), sentry.NewHub(client, sentry.NewScope()))
	handler := sentryslog.Option{}.NewSentryHandler(ctx)
	return output.New(&clientHandler{Handler: handler, client: client}), nil
}

type clientHandler struct {
	slog.Handler
	client *sentry.Client
}

func (h *clientHandler) Handle(ctx context.Context, r slog.Record) error {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil || hub.Client() != h.client {
		ctx = sentry.SetHubOnContext(ctx, sentry.NewHub(h.client, sentry.NewScope()))
	}
	return h.Handler.Handle(ctx, r)
}

func (h *clientHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &clientHandler{Handler: h.Handler.WithAttrs(attrs), client: h.client}
}
func (h *clientHandler) WithGroup(name string) slog.Handler {
	return &clientHandler{Handler: h.Handler.WithGroup(name), client: h.client}
}
