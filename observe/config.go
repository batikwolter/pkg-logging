package observe

import (
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	FormatJSON   = "json"
	OutputStdout = "stdout"
	OutputStderr = "stderr"

	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
)

// Config defines logger behavior.
type Config struct {
	Resource   Resource
	Level      string
	Format     string
	Output     string
	AddSource  bool
	RedactKeys []string
	// GenerateTraceID controls fallback generation when trace.id is missing.
	// Nil means default behavior (enabled).
	GenerateTraceID *bool
	Writer          io.Writer
}

func defaultConfig() Config {
	return Config{
		Level:      LevelInfo,
		Format:     FormatJSON,
		Output:     OutputStdout,
		RedactKeys: DefaultRedactKeys(),
	}
}

func (c Config) withDefaults() Config {
	d := defaultConfig()

	if c.Level == "" {
		c.Level = d.Level
	}
	if c.Format == "" {
		c.Format = d.Format
	}
	if c.Output == "" {
		c.Output = d.Output
	}
	if len(c.RedactKeys) == 0 {
		c.RedactKeys = d.RedactKeys
	}
	return c
}

func (c Config) validate() error {
	level := strings.ToLower(c.Level)
	switch level {
	case LevelDebug, LevelInfo, LevelWarn, LevelError:
	default:
		return fmt.Errorf("invalid level %q", c.Level)
	}

	format := strings.ToLower(c.Format)
	if format != FormatJSON {
		return fmt.Errorf("invalid format %q: only json supported", c.Format)
	}

	if c.Writer == nil {
		output := strings.ToLower(c.Output)
		switch output {
		case OutputStdout, OutputStderr:
		default:
			return fmt.Errorf("invalid output %q", c.Output)
		}
	}

	if strings.TrimSpace(c.Resource.ServiceName) == "" {
		return fmt.Errorf("resource.service_name is required")
	}

	if strings.TrimSpace(c.Resource.Environment) == "" {
		return fmt.Errorf("resource.environment is required")
	}

	return nil
}

func (c Config) levelValue() string {
	return strings.ToLower(c.Level)
}

func (c Config) writer() io.Writer {
	if c.Writer != nil {
		return c.Writer
	}
	if strings.EqualFold(c.Output, OutputStderr) {
		return os.Stderr
	}
	return os.Stdout
}

// BoolPtr returns a pointer to a bool for config literals.
func BoolPtr(v bool) *bool {
	return &v
}

func (c Config) shouldGenerateTraceID() bool {
	if c.GenerateTraceID == nil {
		return true
	}
	return *c.GenerateTraceID
}
