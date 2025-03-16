package metrics_test

import (
	"context"
	"errors"
	"testing"
	"time"

	metricWrapper "github.com/janduursma/otel-metrics-wrapper-go"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestExternalMetrics(t *testing.T) {
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

	// Construct ExternalMetrics.
	em, err := metricWrapper.NewExternalMetrics(meter)
	require.NoError(t, err, "unexpected error creating ExternalMetrics")

	// Simulate a successful external call.
	startSuccess := time.Now()
	em.RecordExternalCall(ctx, "auth-service", "POST")
	em.FinishExternalCall(ctx, "auth-service", "POST", nil, startSuccess)

	// Simulate a failing external call.
	startFail := time.Now()
	simErr := errors.New("simulated external error")
	em.RecordExternalCall(ctx, "payment-service", "GET")
	em.FinishExternalCall(ctx, "payment-service", "GET", simErr, startFail)

	// Force a metrics collection.
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm), "failed to collect metrics")

	// Assert that "external.calls.total" equals 2 (one for each call).
	totalCalls := findIntSumByName(t, rm, "external.calls.total")
	require.EqualValues(t, 2, totalCalls, "expected 2 external calls recorded")

	// Assert that "external.calls.errors" equals 1 (only for the failing call).
	errorCalls := findIntSumByName(t, rm, "external.calls.errors")
	require.EqualValues(t, 1, errorCalls, "expected 1 external call error recorded")

	// Assert that "external.calls.duration" histogram has 2 data points.
	durationCount := findHistogramCountByName(t, rm, "external.calls.duration")
	require.EqualValues(t, 2, durationCount, "expected 2 duration records")
}
