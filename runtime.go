package metrics

import (
	"context"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/metric"
)

// RuntimeMetrics holds the asynchronous gauges for
// goroutines, memory usage, process uptime, etc.
type RuntimeMetrics struct {
	goroutines    metric.Int64ObservableGauge
	memoryHeap    metric.Int64ObservableGauge
	processUptime metric.Int64ObservableGauge

	startTime time.Time
}

// NewRuntimeMetrics creates and registers asynchronous gauges that capture common
// runtime metrics such as the number of goroutines, memory heap usage, and process
// uptime. It returns a struct holding references to these instruments, and also
// registers a callback that the OpenTelemetry SDK periodically invokes to sample
// their values.
func NewRuntimeMetrics(meter metric.Meter) (*RuntimeMetrics, error) {
	rm := &RuntimeMetrics{
		startTime: time.Now(),
	}
	var err error

	rm.goroutines, err = meter.Int64ObservableGauge("go.goroutines")
	if err != nil {
		return nil, err
	}
	rm.memoryHeap, err = meter.Int64ObservableGauge("go.mem.heap_alloc")
	if err != nil {
		return nil, err
	}
	rm.processUptime, err = meter.Int64ObservableGauge("process.uptime")
	if err != nil {
		return nil, err
	}

	// Register a single callback for all three metrics.
	_, err = meter.RegisterCallback(
		// This callback will be called once per collection interval.
		func(_ context.Context, obs metric.Observer) error {
			// Process goroutines.
			obs.ObserveInt64(rm.goroutines, int64(runtime.NumGoroutine()))

			// Process memory.
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			obs.ObserveInt64(rm.memoryHeap, int64(mem.HeapAlloc))

			// Process uptime.
			uptimeSec := int64(time.Since(rm.startTime).Seconds())
			obs.ObserveInt64(rm.processUptime, uptimeSec)

			return nil
		},
		rm.goroutines, rm.memoryHeap, rm.processUptime,
	)
	if err != nil {
		return nil, err
	}

	return rm, nil
}
