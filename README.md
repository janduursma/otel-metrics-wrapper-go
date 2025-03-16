# otel-metrics-wrapper-go

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/janduursma/otel-metrics-wrapper-go)](https://goreportcard.com/report/github.com/janduursma/otel-metrics-wrapper-go)
[![semantic-release](https://img.shields.io/badge/semantic--release-ready-brightgreen)](https://github.com/go-semantic-release/go-semantic-release)
[![codecov](https://codecov.io/gh/janduursma/otel-metrics-wrapper-go/graph/badge.svg?token=LBQGOP14WJ)](https://codecov.io/gh/janduursma/otel-metrics-wrapper-go)

This package provides an instrumentation library for Go microservices, 
built on the OpenTelemetry metrics API. It lets you easily track key statistics in your service
– such as HTTP requests, database calls, external API interactions, and runtime data
– and export them to an OTLP endpoint or other compatible backends.

---

## Features
- **OTLP-Only:** Pushes metrics to an OTLP collector (e.g., SigNoz, OpenTelemetry Collector).
- **Synchronous & Asynchronous Instruments:** Counters, histograms, and gauges for real-time stats
- **HTTP, DB, and External Call Metrics:** Out-of-the-box instrumentation for request tracking, concurrency, error counts, latencies, etc.
- **Runtime Metrics:** Observe goroutines, memory usage, and process uptime.
- **Customizable Histogram Buckets:** Override default aggregator boundaries as needed.
- **Flexible Error Categorization:** A classifyError pattern for capturing timeouts, invalid input, database errors, etc.

---

## Installation
```bash
go get github.com/janduursma/otel-metrics-wrapper-go
```

Then import it in your Go code:

```go
import metrics "github.com/janduursma/otel-metrics-wrapper-go"
```

---

## Configuration
This secrets manager wrapper uses functional options to allow you to customize its behavior. By default, it is configured as follows:
- **Push Interval:** `10 seconds`  
    The default push interval is set to `10 seconds`. You can override this using the `WithPushInterval` option.
- **OTLP Insecure:** `true`  
  The default option to use a secure or insecure option is set to `true` (insecure). You can override this using the `WithOTLPInsecure` option.
- **OTLP CA file:** `""`  
    The default option to specify the path to a CA file is set to `""`. You can override this using the `WithOTLPCAFile` option. Make sure to set `WithOTLPInsecure` to `false` if you provide a CA file.
- **Custom Histogram Views:** `nil`  
    The default option to create any custom histogram buckets is set to `nil`. You can override this using the `WithCustomHistogramViews` option.

---

## Usage
### Create a Config
Configure your OTLP endpoint and other details:
```go
import (
    "time"
    metrics "github.com/janduursma/otel-metrics-wrapper-go"
)

func main() {
    cfg := metrics.Config{
        OTLPEndpoint: "localhost:4317",
        ServiceName:  "my-service",
        Environment:  "prod",
        // Optional
        PushInterval: 20 * time.Second,
    }
    // ...
}
```

### Initialize metrics
Initialize the global MeterProvider and exporter:
```go
if err := metrics.InitMetrics(context.Background(), cfg); err != nil {
    log.Fatalf("failed to init metrics: %v", err)
}
defer func() {
    // Gracefully shutdown on exit
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    metrics.ShutdownMetrics(shutdownCtx)
}()
```

### Construct metrics sets
Create specialized metric sets (HTTP, DB, etc.) after the global provider is ready:
```go
meter := metrics.GetMeter("my-service")

httpMetrics, err := metrics.NewHTTPMetrics(meter)
if err != nil { /* handle error */ }

dbMetrics, err := metrics.NewDBMetrics(meter)
if err != nil { /* handle error */ }

// Repeat for external calls, runtime, etc.
runtimeMetrics, err := metrics.NewRuntimeMetrics(meter)
if err != nil { /* handle error */ }
```

### Record metrics
Use the instrumented methods in your request handlers, DB wrappers, or external clients:
```go
func myHandler(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    httpMetrics.RecordRequestStart(r.Context(), r.Method, "/users")

    defer func() {
        statusCode := 200
        respSize := int64(1234)
        httpMetrics.RecordRequestEnd(r.Context(), r.Method, "/users", statusCode, respSize, start)
    }()

    // handle the request
}

// DB example:
func queryDB(ctx context.Context) {
    start := time.Now()
    dbMetrics.RecordDBCall(ctx, "postgres", "SELECT", "users")
    err := doSomeQuery()
    dbMetrics.FinishDBCall(ctx, "postgres", "SELECT", "users", err, start)
}
```

---

## Instrumentation overview
### HTTPMetrics
- **RequestsTotal:** Counts all HTTP requests.  
- **RequestsErrors:** Counts requests that returned status ≥ 400.
- **RequestsDuration:** Records request latency via an Int64Histogram.
- **RequestsInFlight:** An asynchronous gauge for concurrency.  

Call RecordRequestStart and RecordRequestEnd in your HTTP handlers.

### DBMetrics
- **CallsTotal / CallsErrors:** Tracks DB queries and errors.
- **CallsDuration:** Histogram of query times.  

Use RecordDBCall and FinishDBCall around your DB operations.

### ExternalMetrics
- **CallsTotal / CallsErrors:** Outbound calls to other services.
- **CallsLatency:** Latency histogram.

Wrap your external calls with RecordExternalCall/FinishExternalCall.

### RuntimeMetrics
- **Goroutines:** Number of goroutines.
- **MemoryHeap:** Current heap usage.
- **ProcessUptime:** Time since process start.  

All reported via asynchronous gauges.

---

## Running Tests
```sh
go test ./... -tags test
```

---

## License
- [MIT License](LICENSE)