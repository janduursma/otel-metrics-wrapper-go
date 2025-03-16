package metrics_test

import (
	"context"
	"testing"
	"time"

	metricWrapper "github.com/janduursma/otel-metrics-wrapper-go"
	"github.com/stretchr/testify/require"
	sdkMetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// TestHTTPMetrics_Success tests that a successful HTTP request is recorded correctly.
func TestHTTPMetrics_Success(t *testing.T) {
	// Reset global state so that 'initialized' is false and 'meterProvider' is nil.
	metricWrapper.ResetState()

	ctx := context.Background()

	// Create a ManualReader to collect metrics on demand.
	reader := sdkMetric.NewManualReader()

	// Build a MeterProvider with this ManualReader.
	mp := sdkMetric.NewMeterProvider(
		sdkMetric.WithReader(reader),
	)

	// Acquire a Meter from the MeterProvider.
	meter := mp.Meter("test-meter")

	// Construct HTTPMetrics.
	hm, err := metricWrapper.NewHTTPMetrics(meter)
	require.NoError(t, err, "unexpected error creating HTTPMetrics.")

	// Simulate a successful HTTP request.
	start := time.Now()
	hm.RecordRequestStart(ctx, "GET", "/users")
	// Wait a bit so we have a measurable duration.
	time.Sleep(10 * time.Millisecond)
	hm.RecordRequestEnd(ctx, "GET", "/users", 200, 512, start)

	// Force metrics collection.
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err, "Failed to collect metrics.")

	// Assert that the "requests.total" metric was incremented.
	total := findIntSumByName(t, rm, "requests.total")
	require.EqualValues(t, 1, total, "expected 1 total request.")

	// Assert that the "requests.duration" metric was incremented.
	durationCount := findHistogramCountByName(t, rm, "requests.duration")
	require.EqualValues(t, 1, durationCount, "expected 1 duration record.")

	// Assert that the "response.size" histogram has 1 data point.
	respSizeCount := findHistogramCountByName(t, rm, "response.size")
	require.EqualValues(t, 1, respSizeCount, "expected 1 response size record.")

	// Assert that the asynchronous gauge for in-flight requests is 0.
	inFlight := findGaugeValueByName(t, rm, "requests.in_flight")
	require.EqualValues(t, 0, inFlight, "expected in-flight gauge to be 0.")
}

// TestHTTPMetrics_Error tests that an HTTP request that results in an error
// records an error count while still recording the duration and response size.
func TestHTTPMetrics_Error(t *testing.T) {
	// Reset global state so that 'initialized' is false and 'meterProvider' is nil.
	metricWrapper.ResetState()

	ctx := context.Background()

	// Create a ManualReader to collect metrics on demand.
	reader := sdkMetric.NewManualReader()

	// Build a MeterProvider with this ManualReader.
	mp := sdkMetric.NewMeterProvider(
		sdkMetric.WithReader(reader),
	)

	// Acquire a Meter from the MeterProvider.
	meter := mp.Meter("test-meter")

	// Construct HTTPMetrics.
	hm, err := metricWrapper.NewHTTPMetrics(meter)
	require.NoError(t, err, "failed to create HTTPMetrics")

	// Simulate an HTTP request that results in an error (status code 500).
	start := time.Now()
	hm.RecordRequestStart(ctx, "POST", "/login")
	// Simulate a short processing time.
	time.Sleep(5 * time.Millisecond)
	hm.RecordRequestEnd(ctx, "POST", "/login", 500, 1024, start)

	// Force metrics collection.
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err, "failed to collect metrics.")

	// Assert total requests was incremented.
	total := findIntSumByName(t, rm, "requests.total")
	require.EqualValues(t, 1, total, "expected 1 total request.")

	// Assert error requests was incremented.
	errTotal := findIntSumByName(t, rm, "requests.errors")
	require.EqualValues(t, 1, errTotal, "expected 1 error request.")

	// Assert duration and response size histograms have 1 data point each.
	durationCount := findHistogramCountByName(t, rm, "requests.duration")
	require.EqualValues(t, 1, durationCount, "expected 1 duration record.")
	respSizeCount := findHistogramCountByName(t, rm, "response.size")
	require.EqualValues(t, 1, respSizeCount, "expected 1 response size record.")

	// Assert that the asynchronous gauge for in-flight requests is 0.
	inFlight := findGaugeValueByName(t, rm, "requests.in_flight")
	require.EqualValues(t, 0, inFlight, "expected in-flight gauge to be 0.")
}
