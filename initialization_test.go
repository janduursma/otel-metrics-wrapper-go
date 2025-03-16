package metrics_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	metricWrapper "github.com/janduursma/otel-metrics-wrapper-go"
	"github.com/stretchr/testify/require"
	apimetric "go.opentelemetry.io/otel"
)

func TestInitMetrics_Success(t *testing.T) {
	// Reset global state so that nothing is initialized.
	metricWrapper.ResetState()

	ctx := context.Background()

	// Create a valid config using NewConfig with required parameters.
	cfg := metricWrapper.NewConfig(
		"localhost:4317", // OTLPEndpoint (required)
		"test-service",   // ServiceName (required)
		"test",           // Environment (required)
		metricWrapper.WithPushInterval(10*time.Second),
		metricWrapper.WithOTLPInsecure(true),
		metricWrapper.WithCustomHistogramViews([]metricWrapper.InstrumentViewConfig{
			{
				InstrumentName: "test.histogram",
				Buckets:        []float64{1, 2, 3},
			},
		}),
	)

	// Initialize the metrics pipeline.
	err := metricWrapper.InitMetrics(ctx, cfg)
	require.NoError(t, err, "expected no error during InitMetrics")

	// Get a meter from the global provider.
	m := metricWrapper.GetMeter("dummy")
	require.NotNil(t, m, "expected non-nil meter after initialization")

	// Shutdown the metrics pipeline.
	// Set environment variable to skip flushing during shutdown.
	err = os.Setenv("METRICS_SKIP_FLUSH", "1")
	require.NoError(t, err, "expected no error setting environment variable")

	err = metricWrapper.ShutdownMetrics(ctx)
	require.NoError(t, err, "expected no error during ShutdownMetrics")

	err = os.Unsetenv("METRICS_SKIP_FLUSH")
	require.NoError(t, err, "expected no error unsetting environment variable")
}

func TestInitMetrics_InvalidConfig(t *testing.T) {
	// Reset global state so that nothing is initialized.
	metricWrapper.ResetState()

	ctx := context.Background()

	// Create an invalid config: missing OTLPEndpoint (required).
	cfg := metricWrapper.NewConfig(
		"", // empty endpoint should trigger validation error
		"test-service",
		"test",
	)

	// Expect InitMetrics to return an error.
	err := metricWrapper.InitMetrics(ctx, cfg)
	require.Error(t, err, "expected error due to missing OTLPEndpoint")

	// Reset global state so that nothing is initialized.
	metricWrapper.ResetState()

	// Create an invalid config: missing serviceName (required).
	cfg = metricWrapper.NewConfig(
		"localhost:4317",
		"", // empty serviceName should trigger validation error
		"test",
	)

	// Expect InitMetrics to return an error.
	err = metricWrapper.InitMetrics(ctx, cfg)
	require.Error(t, err, "expected error due to missing serviceName")

	// Reset global state so that nothing is initialized.
	metricWrapper.ResetState()

	// Create an invalid config: missing environment (required).
	cfg = metricWrapper.NewConfig(
		"localhost:4317",
		"test-service",
		"", // empty environment should trigger validation error
	)

	// Expect InitMetrics to return an error.
	err = metricWrapper.InitMetrics(ctx, cfg)
	require.Error(t, err, "expected error due to missing environment")

	// Reset global state so that nothing is initialized.
	metricWrapper.ResetState()

	// Create an invalid config: invalid pushInterval.
	cfg = metricWrapper.NewConfig(
		"localhost:4317",
		"test-service",
		"test",
		metricWrapper.WithPushInterval(0*time.Second), // invalid pushInterval of 0s
	)

	// Expect InitMetrics to return an error.
	err = metricWrapper.InitMetrics(ctx, cfg)
	require.Error(t, err, "expected error due to invalid pushInterval")

	// Reset global state so that nothing is initialized.
	metricWrapper.ResetState()

	// Create an invalid config: specifying secure mode without CA File.
	cfg = metricWrapper.NewConfig(
		"localhost:4317",
		"test-service",
		"test",
		metricWrapper.WithPushInterval(10*time.Second),
		metricWrapper.WithOTLPInsecure(false),
	)

	// Expect InitMetrics to return an error.
	err = metricWrapper.InitMetrics(ctx, cfg)
	require.Error(t, err, "expected error due to missing OTLPCAFile")

	// Reset global state so that nothing is initialized.
	metricWrapper.ResetState()

	// Create an invalid config: specifying invalid CA File.
	cfg = metricWrapper.NewConfig(
		"localhost:4317",
		"test-service",
		"test",
		metricWrapper.WithPushInterval(10*time.Second),
		metricWrapper.WithOTLPInsecure(false),
		metricWrapper.WithOTLPCAFile(""),
	)

	// Expect InitMetrics to return an error.
	err = metricWrapper.InitMetrics(ctx, cfg)
	require.Error(t, err, "expected error due to missing OTLPCAFile")

	// Reset global state so that nothing is initialized.
	metricWrapper.ResetState()

	// Create an invalid config: specifying custom histogram views with invalid instrument name.
	cfg = metricWrapper.NewConfig(
		"localhost:4317",
		"test-service",
		"test",
		metricWrapper.WithPushInterval(10*time.Second),
		metricWrapper.WithOTLPInsecure(true),
		metricWrapper.WithCustomHistogramViews([]metricWrapper.InstrumentViewConfig{
			{
				InstrumentName: "", // empty instrument name should trigger validation error
				Buckets:        []float64{1, 2, 3},
			},
		}),
	)

	// Expect InitMetrics to return an error.
	err = metricWrapper.InitMetrics(ctx, cfg)
	require.Error(t, err, "expected error due to invalid custom histogram views")

	// Reset global state so that nothing is initialized.
	metricWrapper.ResetState()

	// Create an invalid config: specifying custom histogram views with invalid bucket size.
	cfg = metricWrapper.NewConfig(
		"localhost:4317",
		"test-service",
		"test",
		metricWrapper.WithPushInterval(10*time.Second),
		metricWrapper.WithOTLPInsecure(true),
		metricWrapper.WithCustomHistogramViews([]metricWrapper.InstrumentViewConfig{
			{
				InstrumentName: "test-service",
				Buckets:        []float64{1}, // invalid bucket size should trigger validation error
			},
		}),
	)

	// Expect InitMetrics to return an error.
	err = metricWrapper.InitMetrics(ctx, cfg)
	require.Error(t, err, "expected error due to invalid custom histogram views")
}

func TestInitMetrics_SecureInvalidCA(t *testing.T) {
	// Reset global state so that nothing is initialized.
	metricWrapper.ResetState()

	ctx := context.Background()

	// Create a config with OTLPInsecure set to false so the else branch is used,
	// and provide an invalid CA file path.
	cfg := metricWrapper.NewConfig(
		"localhost:4317", // OTLPEndpoint (required)
		"test-service",   // ServiceName (required)
		"test",           // Environment (required)
		metricWrapper.WithOTLPInsecure(false),
		metricWrapper.WithOTLPCAFile("nonexistent-ca.pem"),
	)

	// When initializing, the exporter creation should try to load the CA file
	// and fail, returning an error.
	err := metricWrapper.InitMetrics(ctx, cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to load CA file")
}

func TestShutdownMetrics_Idempotent(t *testing.T) {
	// Reset global state so that nothing is initialized.
	metricWrapper.ResetState()

	ctx := context.Background()

	// Create a valid config.
	cfg := metricWrapper.NewConfig(
		"localhost:4317",
		"test-service",
		"test",
		metricWrapper.WithPushInterval(1*time.Hour),
		metricWrapper.WithOTLPInsecure(true),
	)
	err := metricWrapper.InitMetrics(ctx, cfg)
	require.NoError(t, err, "expected no error during InitMetrics")

	// Call ShutdownMetrics once.
	err = os.Setenv("METRICS_SKIP_FLUSH", "1")
	require.NoError(t, err, "expected no error setting environment variable")

	err = metricWrapper.ShutdownMetrics(ctx)
	require.NoError(t, err, "expected no error on first shutdown")

	err = os.Unsetenv("METRICS_SKIP_FLUSH")
	require.NoError(t, err, "expected no error unsetting environment variable")

	// Call ShutdownMetrics a second time; it should be idempotent.
	err = os.Setenv("METRICS_SKIP_FLUSH", "1")
	require.NoError(t, err, "expected no error setting environment variable")

	err = metricWrapper.ShutdownMetrics(ctx)
	require.NoError(t, err, "expected no error on second shutdown")

	err = os.Unsetenv("METRICS_SKIP_FLUSH")
	require.NoError(t, err, "expected no error unsetting environment variable")
}

func TestNewConfigDefaults(t *testing.T) {
	// Reset global state so that nothing is initialized.
	metricWrapper.ResetState()

	// NewConfig is expected to set optional fields to defaults.
	cfg := metricWrapper.NewConfig("localhost:4317", "test-service", "test")
	require.Equal(t, 10*time.Second, cfg.PushInterval, "expected default push interval of 10s")
	require.True(t, cfg.OTLPInsecure, "expected default OTLPInsecure to be true")
	require.Equal(t, "", cfg.OTLPCAFile, "expected default OTLPCAFile to be empty")
	require.Nil(t, cfg.CustomHistogramViews, "expected default CustomHistogramViews to be nil")
}

func TestShutdownMetrics_NotInitialized(t *testing.T) {
	// Reset global state so that nothing is initialized.
	metricWrapper.ResetState()

	// Now call ShutdownMetrics. The closure should run, detect that 'initialized' is false,
	// and return early.
	err := metricWrapper.ShutdownMetrics(context.Background())
	require.NoError(t, err, "ShutdownMetrics should return nil when not initialized")
}

// GetMeter returns a Meter from the global (default) provider.
func TestGetMeter_Uninitialized(t *testing.T) {
	// Reset global state so that 'initialized' is false and 'meterProvider' is nil.
	metricWrapper.ResetState()

	// Call GetMeter, which should take the uninitialized branch.
	m := metricWrapper.GetMeter("test-meter")
	require.NotNil(t, m, "expected a non-nil meter from the default provider")

	// For further verification, obtain a meter directly from the default provider.
	defaultMeter := apimetric.GetMeterProvider().Meter("test-meter")
	// We cannot directly compare interfaces for equality,
	// but we can check that their types match.
	require.Equal(t, fmt.Sprintf("%T", defaultMeter), fmt.Sprintf("%T", m),
		"expected GetMeter to return a meter of the same type as the default provider")
}
