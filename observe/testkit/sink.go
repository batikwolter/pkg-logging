package testkit

import (
	"bytes"
	"encoding/json"
	"sync"
)

// Sink captures newline-delimited JSON logs in-memory.
type Sink struct {
	mu      sync.Mutex
	pending []byte
	entries []map[string]any
}

func NewSink() *Sink {
	return &Sink{}
}

func (s *Sink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pending = append(s.pending, p...)
	for {
		idx := bytes.IndexByte(s.pending, '\n')
		if idx < 0 {
			break
		}
		line := bytes.TrimSpace(s.pending[:idx])
		s.pending = s.pending[idx+1:]
		if len(line) == 0 {
			continue
		}
		entry := map[string]any{}
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		s.entries = append(s.entries, entry)
	}
	return len(p), nil
}

func (s *Sink) Entries() []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]map[string]any, len(s.entries))
	for i, entry := range s.entries {
		cloned := make(map[string]any, len(entry))
		for key, value := range entry {
			cloned[key] = value
		}
		out[i] = cloned
	}
	return out
}

func (s *Sink) Last() (map[string]any, bool) {
	entries := s.Entries()
	if len(entries) == 0 {
		return nil, false
	}
	return entries[len(entries)-1], true
}
