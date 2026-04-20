package observe

import (
	"time"
)

// Field is an org-owned logging field contract.
type Field struct {
	Key        string
	Value      any
	Sensitive  bool
	MetricSafe bool
}

func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value}
}

func Time(key string, value time.Time) Field {
	return Field{Key: key, Value: value}
}

func Error(key string, err error) Field {
	if err == nil {
		return Field{Key: key, Value: ""}
	}
	return Field{Key: key, Value: err.Error()}
}

func SensitiveString(key, value string) Field {
	return Field{Key: key, Value: value, Sensitive: true}
}
