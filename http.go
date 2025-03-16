package metrics

import (
	"context"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// HTTPMetrics holds all instruments for HTTP requests.
type HTTPMetrics struct {
	// Synchronous instruments.
	RequestsTotal    metric.Int64Counter
	RequestsErrors   metric.Int64Counter
	RequestsDuration metric.Int64Histogram
	ResponseSize     metric.Int64Histogram

	// Asynchronous gauge for concurrency.
	RequestsInFlight metric.Int64ObservableGauge

	// Atomic for concurrency tracking.
	inFlight int64
}

// NewHTTPMetrics creates and registers a set of instruments designed for HTTP
// request tracking, including total and error counters, request duration and
// response size histograms, and an asynchronous gauge for in-flight requests.
// It returns a struct holding references to these instruments, and also registers
// a callback that periodically captures the current concurrency level.
func NewHTTPMetrics(meter metric.Meter) (*HTTPMetrics, error) {
	hm := &HTTPMetrics{}
	var err error

	// Create synchronous instruments.
	if hm.RequestsTotal, err = meter.Int64Counter("requests.total"); err != nil {
		return nil, err
	}
	if hm.RequestsErrors, err = meter.Int64Counter("requests.errors"); err != nil {
		return nil, err
	}
	if hm.RequestsDuration, err = meter.Int64Histogram("requests.duration"); err != nil {
		return nil, err
	}
	if hm.ResponseSize, err = meter.Int64Histogram("response.size"); err != nil {
		return nil, err
	}

	// Create an asynchronous gauge for concurrency.
	if hm.RequestsInFlight, err = meter.Int64ObservableGauge("requests.in_flight"); err != nil {
		return nil, err
	}

	// Register a callback that the SDK will call periodically.
	// It reads the atomic inFlight counter and observes it.
	_, err = meter.RegisterCallback(
		func(_ context.Context, obs metric.Observer) error {
			current := atomic.LoadInt64(&hm.inFlight)
			obs.ObserveInt64(hm.RequestsInFlight, current)
			return nil
		},
		// List all instruments observed in this callback.
		hm.RequestsInFlight,
	)
	if err != nil {
		return nil, err
	}

	return hm, nil
}

// RecordRequestStart increments the total requests counter & concurrency.
func (hm *HTTPMetrics) RecordRequestStart(ctx context.Context, method, route string) {
	hm.RequestsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("route", route),
		),
	)
	atomic.AddInt64(&hm.inFlight, 1)
}

// RecordRequestEnd decrements concurrency, records errors, latency, etc.
func (hm *HTTPMetrics) RecordRequestEnd(
	ctx context.Context,
	method, route string,
	statusCode int,
	respSize int64,
	start time.Time,
) {
	atomic.AddInt64(&hm.inFlight, -1)

	// Record error if status code is 4xx or 5xx.
	if statusCode >= 400 {
		hm.RequestsErrors.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("method", method),
				attribute.String("route", route),
				attribute.Int("status_code", statusCode),
			),
		)
	}

	// Record request latency as elapsed milliseconds.
	elapsedMs := time.Since(start).Milliseconds()
	hm.RequestsDuration.Record(ctx, elapsedMs,
		metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("route", route),
			attribute.Int("status_code", statusCode),
		),
	)

	// Record response size.
	hm.ResponseSize.Record(ctx, respSize,
		metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("route", route),
			attribute.Int("status_code", statusCode),
		),
	)
}
