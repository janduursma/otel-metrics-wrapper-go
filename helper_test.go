package metrics_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// findGaugeValueByName scans the ResourceMetrics for a gauge metric with the given name
// and returns the value of its first data point.
func findGaugeValueByName(t *testing.T, rm metricdata.ResourceMetrics, name string) int64 {
	var value int64
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				// Try int64 gauge first.
				if gauge, ok := m.Data.(metricdata.Gauge[int64]); ok {
					if len(gauge.DataPoints) > 0 {
						value = gauge.DataPoints[0].Value
						found = true
					}
				} else if gauge, ok := m.Data.(metricdata.Gauge[float64]); ok {
					if len(gauge.DataPoints) > 0 {
						value = int64(gauge.DataPoints[0].Value)
						found = true
					}
				}
			}
		}
	}
	require.True(t, found, "gauge metric %q not found", name)
	return value
}

// findHistogramCountByName scans the ResourceMetrics for a histogram metric with the given name
// and returns the sum of counts for all its data points.
// It handles both int64 and float64 histograms.
func findHistogramCountByName(t *testing.T, rm metricdata.ResourceMetrics, name string) uint64 {
	var total uint64
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				// Try int64 histogram.
				if histInt, ok := m.Data.(metricdata.Histogram[int64]); ok {
					for _, dp := range histInt.DataPoints {
						total += dp.Count
					}
					found = true
				}
				// Try float64 histogram.
				if histFloat, ok := m.Data.(metricdata.Histogram[float64]); ok {
					for _, dp := range histFloat.DataPoints {
						total += dp.Count
					}
					found = true
				}
			}
		}
	}
	require.True(t, found, "histogram metric %q not found", name)
	return total
}

// findIntSumByName scans through the ResourceMetrics for all Sum[int64] metrics
// with the specified name and sums the values of all its data points.
func findIntSumByName(t *testing.T, rm metricdata.ResourceMetrics, name string) int64 {
	var total int64
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				sum, ok := m.Data.(metricdata.Sum[int64])
				require.True(t, ok, "expected Sum[int64] for metric %q", name)
				for _, dp := range sum.DataPoints {
					total += dp.Value
				}
				found = true
			}
		}
	}
	require.True(t, found, "metric %q not found in ResourceMetrics", name)
	return total
}
