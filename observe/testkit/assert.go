package testkit

import (
	"fmt"
	"testing"
)

func RequireEntries(t testing.TB, sink *Sink, min int) []map[string]any {
	t.Helper()
	entries := sink.Entries()
	if len(entries) < min {
		t.Fatalf("expected at least %d log entries, got %d", min, len(entries))
	}
	return entries
}

func FindEntryByEvent(entries []map[string]any, event string) (map[string]any, bool) {
	for _, entry := range entries {
		if fmt.Sprint(entry["event"]) == event {
			return entry, true
		}
	}
	return nil, false
}

func RequireFieldEqual(t testing.TB, entry map[string]any, key string, expected any) {
	t.Helper()
	value, ok := entry[key]
	if !ok {
		t.Fatalf("missing field %q", key)
	}
	if fmt.Sprint(value) != fmt.Sprint(expected) {
		t.Fatalf("field %q mismatch: got %v want %v", key, value, expected)
	}
}

func RequireFieldNotEmpty(t testing.TB, entry map[string]any, key string) {
	t.Helper()
	value, ok := entry[key]
	if !ok {
		t.Fatalf("missing field %q", key)
	}
	if fmt.Sprint(value) == "" {
		t.Fatalf("field %q is empty", key)
	}
}
