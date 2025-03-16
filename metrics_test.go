package metrics_test

import (
	"fmt"
	"testing"

	metricWrapper "github.com/janduursma/otel-metrics-wrapper-go"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

// fakeMeter embeds a real metric.Meter and overrides selected methods
// to force errors for a specific instrument name.
type fakeMeter struct {
	metric.Meter // Embedded: provides all methods not overridden.
	error        string
}

// Override Int64Counter to force an error if the instrument name matches error.
func (f fakeMeter) Int64Counter(name string, opts ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	if name == f.error {
		return nil, fmt.Errorf("forced error for counter %s", name)
	}
	return f.Meter.Int64Counter(name, opts...)
}

// Override Int64Histogram to force an error if the instrument name matches error.
func (f fakeMeter) Int64Histogram(name string, opts ...metric.Int64HistogramOption) (metric.Int64Histogram, error) {
	if name == f.error {
		return nil, fmt.Errorf("forced error for histogram %s", name)
	}
	return f.Meter.Int64Histogram(name, opts...)
}

// Override Int64ObservableGauge to force an error if the instrument name matches error.
func (f fakeMeter) Int64ObservableGauge(name string, opts ...metric.Int64ObservableGaugeOption) (metric.Int64ObservableGauge, error) {
	if name == f.error {
		return nil, fmt.Errorf("forced error for observable gauge %s", name)
	}
	return f.Meter.Int64ObservableGauge(name, opts...)
}

// For all other methods, delegate to the underlying meter.
func (f fakeMeter) Int64UpDownCounter(name string, opts ...metric.Int64UpDownCounterOption) (metric.Int64UpDownCounter, error) {
	return f.Meter.Int64UpDownCounter(name, opts...)
}

func (f fakeMeter) Int64Gauge(name string, opts ...metric.Int64GaugeOption) (metric.Int64Gauge, error) {
	return f.Meter.Int64Gauge(name, opts...)
}

func (f fakeMeter) Float64Counter(name string, opts ...metric.Float64CounterOption) (metric.Float64Counter, error) {
	return f.Meter.Float64Counter(name, opts...)
}

func (f fakeMeter) Float64Histogram(name string, opts ...metric.Float64HistogramOption) (metric.Float64Histogram, error) {
	return f.Meter.Float64Histogram(name, opts...)
}

func (f fakeMeter) Float64UpDownCounter(name string, opts ...metric.Float64UpDownCounterOption) (metric.Float64UpDownCounter, error) {
	return f.Meter.Float64UpDownCounter(name, opts...)
}

func (f fakeMeter) Float64Gauge(name string, opts ...metric.Float64GaugeOption) (metric.Float64Gauge, error) {
	return f.Meter.Float64Gauge(name, opts...)
}

func (f fakeMeter) Int64ObservableCounter(name string, opts ...metric.Int64ObservableCounterOption) (metric.Int64ObservableCounter, error) {
	return f.Meter.Int64ObservableCounter(name, opts...)
}

func (f fakeMeter) Int64ObservableUpDownCounter(name string, opts ...metric.Int64ObservableUpDownCounterOption) (metric.Int64ObservableUpDownCounter, error) {
	return f.Meter.Int64ObservableUpDownCounter(name, opts...)
}

func (f fakeMeter) Float64ObservableCounter(name string, opts ...metric.Float64ObservableCounterOption) (metric.Float64ObservableCounter, error) {
	return f.Meter.Float64ObservableCounter(name, opts...)
}

func (f fakeMeter) Float64ObservableUpDownCounter(name string, opts ...metric.Float64ObservableUpDownCounterOption) (metric.Float64ObservableUpDownCounter, error) {
	return f.Meter.Float64ObservableUpDownCounter(name, opts...)
}

func (f fakeMeter) Float64ObservableGauge(name string, opts ...metric.Float64ObservableGaugeOption) (metric.Float64ObservableGauge, error) {
	return f.Meter.Float64ObservableGauge(name, opts...)
}

func (f fakeMeter) RegisterCallback(cb metric.Callback, instruments ...metric.Observable) (metric.Registration, error) {
	return f.Meter.RegisterCallback(cb, instruments...)
}

//
// Test suite for NewMetrics
//

// TestNewMetrics_Success verifies that NewMetrics returns a valid Metrics struct
// when all sub-instrument creations succeed.
func TestNewMetrics_Success(t *testing.T) {
	// Use a noop Meter as delegate.
	meter := noop.NewMeterProvider().Meter("noop")
	fm := fakeMeter{error: "", Meter: meter}

	m, err := metricWrapper.NewMetrics(fm)
	require.NoError(t, err, "expected no error from NewMetrics")
	require.NotNil(t, m, "expected non-nil Metrics struct")
	require.NotNil(t, m.HTTP, "expected non-nil HTTP metrics")
	require.NotNil(t, m.DB, "expected non-nil DB metrics")
	require.NotNil(t, m.External, "expected non-nil External metrics")
	require.NotNil(t, m.Runtime, "expected non-nil Runtime metrics")
}

// TestNewMetrics_HTTPError forces an error in HTTP metrics creation.
func TestNewMetrics_HTTPError(t *testing.T) {
	// Force error on "requests.total" which is created in NewHTTPMetrics.
	meter := noop.NewMeterProvider().Meter("noop")
	fm := fakeMeter{error: "requests.total", Meter: meter}

	_, err := metricWrapper.NewMetrics(fm)
	require.Error(t, err, "expected error when HTTP metrics creation fails")
	require.Contains(t, err.Error(), "forced error for counter requests.total")
}

// TestNewMetrics_DBError forces an error in DB metrics creation.
func TestNewMetrics_DBError(t *testing.T) {
	// Force error on "db.calls.total" used in NewDBMetrics.
	meter := noop.NewMeterProvider().Meter("noop")
	fm := fakeMeter{error: "db.calls.total", Meter: meter}

	_, err := metricWrapper.NewMetrics(fm)
	require.Error(t, err, "expected error when DB metrics creation fails")
	require.Contains(t, err.Error(), "forced error for counter db.calls.total")
}

// TestNewMetrics_ExternalError forces an error in External metrics creation.
func TestNewMetrics_ExternalError(t *testing.T) {
	// Force error on "external.calls.total" used in NewExternalMetrics.
	meter := noop.NewMeterProvider().Meter("noop")
	fm := fakeMeter{error: "external.calls.total", Meter: meter}

	_, err := metricWrapper.NewMetrics(fm)
	require.Error(t, err, "expected error when External metrics creation fails")
	require.Contains(t, err.Error(), "forced error for counter external.calls.total")
}

// TestNewMetrics_RuntimeError forces an error in Runtime metrics creation.
func TestNewMetrics_RuntimeError(t *testing.T) {
	// Force error on "go.goroutines" used in NewRuntimeMetrics.
	meter := noop.NewMeterProvider().Meter("noop")
	fm := fakeMeter{error: "go.goroutines", Meter: meter}

	_, err := metricWrapper.NewMetrics(fm)
	require.Error(t, err, "expected error when Runtime metrics creation fails")
	require.Contains(t, err.Error(), "forced error for observable gauge go.goroutines")
}
