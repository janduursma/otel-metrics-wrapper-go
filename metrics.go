/*
Package metrics provides a consistent way to record application-specific
runtime metrics across different service components, using the OpenTelemetry metrics API.
Integrating this package ensures data is captured, aggregated,
and exported reliably to an OpenTelemetry-compatible backend.
*/
package metrics

import (
	"log"

	"go.opentelemetry.io/otel/metric"
)

// Metrics is the top-level struct that holds all categories
// of metrics in a service. Each sub-struct focuses on a category.
type Metrics struct {
	HTTP     *HTTPMetrics
	DB       *DBMetrics
	External *ExternalMetrics
	Runtime  *RuntimeMetrics
}

// NewMetrics constructs all sub-structs and registers
// asynchronous instruments/callbacks with the given Meter.
func NewMetrics(meter metric.Meter) (*Metrics, error) {
	var (
		am  Metrics
		err error
	)

	// Create HTTP metrics
	am.HTTP, err = NewHTTPMetrics(meter)
	if err != nil {
		return nil, err
	}

	// Create DB metrics
	am.DB, err = NewDBMetrics(meter)
	if err != nil {
		return nil, err
	}

	// Create External metrics
	am.External, err = NewExternalMetrics(meter)
	if err != nil {
		return nil, err
	}

	// Create Runtime metrics
	am.Runtime, err = NewRuntimeMetrics(meter)
	if err != nil {
		return nil, err
	}

	log.Println("[metrics] Successfully created all metric instruments.")
	return &am, nil
}
