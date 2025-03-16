package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// ExternalMetrics holds a set of instruments for external metrics.
type ExternalMetrics struct {
	CallsTotal   metric.Int64Counter
	CallsErrors  metric.Int64Counter
	CallsLatency metric.Int64Histogram
}

// NewExternalMetrics creates and registers a set of instruments for tracking
// outbound requests to external services or APIs, including total and error
// counters and a histogram for call latency. It returns a struct that holds
// references to these instruments.
func NewExternalMetrics(meter metric.Meter) (*ExternalMetrics, error) {
	em := &ExternalMetrics{}
	var err error

	if em.CallsTotal, err = meter.Int64Counter("external.calls.total"); err != nil {
		return nil, err
	}
	if em.CallsErrors, err = meter.Int64Counter("external.calls.errors"); err != nil {
		return nil, err
	}
	if em.CallsLatency, err = meter.Int64Histogram("external.calls.duration"); err != nil {
		return nil, err
	}

	return em, nil
}

// RecordExternalCall increments the total calls.
func (em *ExternalMetrics) RecordExternalCall(ctx context.Context, targetService, method string) {
	em.CallsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("target_service", targetService),
			attribute.String("method", method),
		),
	)
}

// FinishExternalCall records the latency and error status of an external call.
func (em *ExternalMetrics) FinishExternalCall(
	ctx context.Context,
	targetService, method string,
	err error,
	start time.Time,
) {
	if err != nil {
		em.CallsErrors.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("target_service", targetService),
				attribute.String("method", method),
				attribute.String("error_type", classifyError(err)),
			),
		)
	}
	elapsedMs := time.Since(start).Milliseconds()
	em.CallsLatency.Record(ctx, elapsedMs,
		metric.WithAttributes(
			attribute.String("target_service", targetService),
			attribute.String("method", method),
			attribute.Bool("error", err != nil),
		),
	)
}
