package output

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
)

type probe struct {
	slog.Handler
	calls   *int
	enabled bool
	fail    bool
	mutate  bool
}

func (p probe) Enabled(context.Context, slog.Level) bool { return p.enabled }
func (p probe) Handle(_ context.Context, r slog.Record) error {
	*p.calls++
	if p.mutate {
		r.AddAttrs(slog.String("extra", "private"))
	} else {
		r.Attrs(func(a slog.Attr) bool {
			if a.Key == "extra" {
				panic("record mutation leaked")
			}
			return true
		})
	}
	if p.fail {
		return errors.New("output failed")
	}
	return nil
}

func TestComposeAttemptsEveryEnabledOutput(t *testing.T) {
	for _, failPrimary := range []bool{true, false} {
		first, second, disabled := 0, 0, 0
		base := slog.NewJSONHandler(io.Discard, nil)
		h := Compose(probe{Handler: base, calls: &first, enabled: true, fail: failPrimary, mutate: true}, slog.LevelInfo, []Destination{
			New(probe{Handler: base, calls: &second, enabled: true, fail: !failPrimary}),
			New(probe{Handler: base, calls: &disabled}),
		})
		if h.Enabled(context.Background(), slog.LevelDebug) {
			t.Fatal("global level ignored")
		}
		if err := h.Handle(context.Background(), slog.Record{Level: slog.LevelInfo}); err == nil {
			t.Fatal("missing output error")
		}
		if first != 1 || second != 1 || disabled != 0 {
			t.Fatalf("calls %d %d %d", first, second, disabled)
		}
	}
}
