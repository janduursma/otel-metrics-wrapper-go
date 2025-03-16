package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// DBMetrics holds instruments for database operations.
type DBMetrics struct {
	CallsTotal    metric.Int64Counter
	CallsErrors   metric.Int64Counter
	CallsDuration metric.Int64Histogram
}

// NewDBMetrics creates and registers a set of instruments for tracking database
// interactions, including total and error counters, along with a histogram for query
// duration. It returns a struct holding references to these instruments.
func NewDBMetrics(meter metric.Meter) (*DBMetrics, error) {
	dbm := &DBMetrics{}
	var err error

	if dbm.CallsTotal, err = meter.Int64Counter("db.calls.total"); err != nil {
		return nil, err
	}
	if dbm.CallsErrors, err = meter.Int64Counter("db.calls.errors"); err != nil {
		return nil, err
	}
	if dbm.CallsDuration, err = meter.Int64Histogram("db.calls.duration"); err != nil {
		return nil, err
	}

	return dbm, nil
}

// RecordDBCall increments the DB calls counter.
func (dbm *DBMetrics) RecordDBCall(ctx context.Context, dbSystem, operation, table string) {
	dbm.CallsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("db_system", dbSystem),
			attribute.String("operation", operation),
			attribute.String("table", table),
		),
	)
}

// FinishDBCall records errors & latency.
func (dbm *DBMetrics) FinishDBCall(
	ctx context.Context,
	dbSystem, operation, table string,
	err error,
	start time.Time,
) {
	if err != nil {
		dbm.CallsErrors.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("db_system", dbSystem),
				attribute.String("operation", operation),
				attribute.String("table", table),
				attribute.String("error_type", classifyError(err)),
			),
		)
	}
	elapsedMs := time.Since(start).Milliseconds()
	dbm.CallsDuration.Record(ctx, elapsedMs,
		metric.WithAttributes(
			attribute.String("db_system", dbSystem),
			attribute.String("operation", operation),
			attribute.String("table", table),
			attribute.Bool("error", err != nil),
		),
	)
}
