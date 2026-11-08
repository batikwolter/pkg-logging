// Package observeotlp bridges logs to an explicit OpenTelemetry LoggerProvider.
// Applications own privacy processors, resources, exporters and provider lifecycle.
package observeotlp

import (
	"errors"
	"reflect"

	"github.com/deccara-tech/pkg-logging/internal/output"
	"github.com/deccara-tech/pkg-logging/observe"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/log"
)

// New builds a destination without changing or selecting global providers.
// Nil and typed-nil providers are rejected. Pass the result in Config.Destinations.
// Field masking does not sanitize messages, nested values or SDK enrichment;
// configure a fail-closed privacy processor before any remote export.
// New never flushes or shuts down provider.
func New(provider log.LoggerProvider) (observe.Destination, error) {
	if provider == nil {
		return observe.Destination{}, errors.New("OTLP logger provider is required")
	}
	value := reflect.ValueOf(provider)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if value.IsNil() {
			return observe.Destination{}, errors.New("OTLP logger provider is required")
		}
	}
	return output.New(otelslog.NewHandler(
		"github.com/deccara-tech/pkg-logging/observe/otlp",
		otelslog.WithLoggerProvider(provider),
	)), nil
}
