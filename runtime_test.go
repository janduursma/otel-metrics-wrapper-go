package metrics_test

import (
	"context"
	"testing"
	"time"

	metricWrapper "github.com/janduursma/otel-metrics-wrapper-go"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// TestRuntimeMetrics verifies that the asynchronous gauges for goroutines,
// memory heap allocation, and process uptime are being recorded by the callback.
func TestRuntimeMetrics(t *testing.T) {
	// Reset global state so that 'initialized' is false and 'meterProvider' is nil.
	metricWrapper.ResetState()

	ctx := context.Background()

	// Create a ManualReader to collect metrics on demand.
	reader := metric.NewManualReader()

	// Build a MeterProvider with this ManualReader.
	mp := metric.NewMeterProvider(
		metric.WithReader(reader),
	)

	// Acquire a Meter from the MeterProvider.
	meter := mp.Meter("test-meter")

	// Construct RuntimeMetrics.
	_, err := metricWrapper.NewRuntimeMetrics(meter)
	require.NoError(t, err, "failed to create RuntimeMetrics.")

	// Allow some time for the callback to run and capture uptime.
	time.Sleep(1 * time.Second)

	// Force a metrics collection.
	var res metricdata.ResourceMetrics
	err = reader.Collect(ctx, &res)
	require.NoError(t, err, "failed to collect metrics.")

	// Assert that the gauges report reasonable (non-zero) values.

	// Assert that at least one goroutine is running.
	gCount := findGaugeValueByName(t, res, "go.goroutines")
	require.GreaterOrEqual(t, gCount, int64(1), "expected at least 1 goroutine, got %d", gCount)

	// Expect some heap allocation.
	heapAlloc := findGaugeValueByName(t, res, "go.mem.heap_alloc")
	require.Greater(t, heapAlloc, int64(0), "expected heap allocation > 0, got %d", heapAlloc)

	// Assert that uptime is greater than 0.
	uptime := findGaugeValueByName(t, res, "process.uptime")
	require.Greater(t, uptime, int64(0), "expected uptime > 0, got %d", uptime)
}
