package observe

import "strings"

const defaultRedactionMask = "[REDACTED]"

type redactor struct {
	keys []string
}

func newRedactor(keys []string) redactor {
	if len(keys) == 0 {
		keys = DefaultRedactKeys()
	}
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		if trimmed := strings.TrimSpace(strings.ToLower(key)); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return redactor{keys: out}
}

func DefaultRedactKeys() []string {
	return []string{
		"authorization",
		"cookie",
		"password",
		"secret",
		"token",
		"api key",
		"api-key",
		"apikey",
	}
}

func (r redactor) sanitize(fields []Field) []Field {
	if len(fields) == 0 {
		return nil
	}
	out := make([]Field, 0, len(fields))
	for _, field := range fields {
		out = append(out, r.sanitizeField(field))
	}
	return out
}

func (r redactor) sanitizeField(field Field) Field {
	if field.Sensitive || r.isSensitiveKey(field.Key) {
		field.Value = defaultRedactionMask
	}
	return field
}

func (r redactor) isSensitiveKey(key string) bool {
	if key == "" {
		return false
	}
	keyLower := strings.ToLower(key)
	keyCompact := strings.NewReplacer("_", "", "-", "", ".", "", " ", "").Replace(keyLower)
	for _, pattern := range r.keys {
		if strings.Contains(keyLower, pattern) {
			return true
		}
		compactPattern := strings.NewReplacer("_", "", "-", "", ".", "", " ", "").Replace(pattern)
		if compactPattern != "" && strings.Contains(keyCompact, compactPattern) {
			return true
		}
	}
	return false
}
