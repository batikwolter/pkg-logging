package output

import (
	"context"
	"io"
	"log/slog"
	"testing"
)

type appendProbe struct {
	slog.Handler
	key   string
	saved *slog.Record
}

func (p appendProbe) Handle(_ context.Context, r slog.Record) error {
	r.AddAttrs(slog.String(p.key, "value"))
	*p.saved = r
	return nil
}

func TestComposeClonesSharedRecordStorage(t *testing.T) {
	base := slog.NewJSONHandler(io.Discard, nil)
	var first, second slog.Record
	h := Compose(appendProbe{base, "first", &first}, slog.LevelInfo, []Destination{New(appendProbe{base, "second", &second})})
	r := slog.Record{Level: slog.LevelInfo}
	// Fill inline attrs and retain spare backing capacity for child appends.
	attrs := make([]slog.Attr, 6, 16)
	for i := range attrs {
		attrs[i] = slog.Int("initial", i)
	}
	r.AddAttrs(attrs...)
	r.AddAttrs(slog.Int("initial", 6), slog.Int("initial", 7))
	r.AddAttrs(slog.Int("initial", 8))
	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		record          slog.Record
		want, forbidden string
	}{{first, "first", "second"}, {second, "second", "first"}} {
		found := false
		item.record.Attrs(func(a slog.Attr) bool {
			if a.Key == "!BUG" {
				t.Error("record copied without cloning")
			}
			if a.Key == item.forbidden {
				t.Errorf("child mutation leaked: %s", a.Key)
			}
			if a.Key == item.want {
				found = true
			}
			return true
		})
		if !found {
			t.Errorf("missing child attribute %s", item.want)
		}
	}
}
