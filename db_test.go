package metrics_test

import (
	"context"
	"errors"
	"testing"
	"time"

	metricWrapper "github.com/janduursma/otel-metrics-wrapper-go"
	"github.com/stretchr/testify/require"

	sdkMetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestDBMetrics(t *testing.T) {
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

	// Construct DBMetrics.
	dbm, err := metricWrapper.NewDBMetrics(meter)
	require.NoError(t, err, "unexpected error creating DBMetrics.")

	// Simulate a DB call.
	simulatedError := errors.New("simulated DB error")
	start := time.Now()
	dbm.FinishDBCall(ctx, "postgres", "INSERT", "users", simulatedError, start)
	dbm.RecordDBCall(ctx, "postgres", "SELECT", "users")
	dbm.FinishDBCall(ctx, "postgres", "SELECT", "users", nil, start)

	// Force a metrics collection.
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err, "failed to collect metrics.")

	// Assert that the "db.calls.total" metric was incremented.
	callsTotal := findIntSumByName(t, rm, "db.calls.total")
	require.EqualValues(t, 1, callsTotal, "expected 1 DB call record.")

	// Assert that the "db.calls.duration" histogram has two data points.
	callsDurationCount := findHistogramCountByName(t, rm, "db.calls.duration")
	require.EqualValues(t, 2, callsDurationCount, "expected two duration records.")

	// Assert that the "db.calls.errors" metric was incremented.
	errorsCount := findIntSumByName(t, rm, "db.calls.errors")
	require.EqualValues(t, 1, errorsCount, "expected one error to be recorded.")
}
