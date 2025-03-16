package metrics

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel/metric"

	apimetric "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Config holds the configuration for the OTLP metrics exporter and MeterProvider.
type Config struct {
	OTLPEndpoint         string
	OTLPInsecure         bool
	OTLPCAFile           string
	PushInterval         time.Duration
	ServiceName          string
	Environment          string
	CustomHistogramViews []InstrumentViewConfig
}

// Option is the function signature for functional options.
type Option func(*Config)

// InstrumentViewConfig holds the configuration for a custom histogram view.
type InstrumentViewConfig struct {
	InstrumentName string
	Buckets        []float64
}

// Global variables for the MeterProvider and shutdown function.
var (
	meterProvider *sdkmetric.MeterProvider
	shutdownFunc  func(context.Context) error
	initOnce      sync.Once
	shutdownOnce  sync.Once
	initialized   bool
	mu            sync.RWMutex
)

// WithPushInterval sets the interval for pushing metrics to the exporter.
func WithPushInterval(interval time.Duration) Option {
	return func(cfg *Config) {
		cfg.PushInterval = interval
	}
}

// WithCustomHistogramViews sets custom histogram views for the MeterProvider.
func WithCustomHistogramViews(views []InstrumentViewConfig) Option {
	return func(cfg *Config) {
		cfg.CustomHistogramViews = views
	}
}

// WithOTLPInsecure sets the OTLP exporter to use a secure or insecure connection.
func WithOTLPInsecure(insecure bool) Option {
	return func(cfg *Config) {
		cfg.OTLPInsecure = insecure
	}
}

// WithOTLPCAFile sets the CA file for secure OTLP connections.
func WithOTLPCAFile(caFile string) Option {
	return func(cfg *Config) {
		cfg.OTLPCAFile = caFile
	}
}

// InitMetrics configures an OTLP gRPC exporter and sets up the global MeterProvider.
func InitMetrics(ctx context.Context, cfg Config) error {
	if err := validateConfig(cfg); err != nil {
		return fmt.Errorf("invalid OTLP metrics config: %w", err)
	}

	var initErr error
	initOnce.Do(func() {
		// Create the OTLP exporter.
		exporter, err := createOTLPExporter(ctx, cfg)
		if err != nil {
			initErr = fmt.Errorf("failed to create OTLP exporter: %w", err)
			return
		}

		// Create a resource to label the service.
		r, err := resource.New(ctx,
			resource.WithHost(),
			resource.WithContainer(),
			resource.WithAttributes(
				semconv.ServiceNameKey.String(cfg.ServiceName),
				semconv.DeploymentEnvironmentKey.String(cfg.Environment),
			),
		)
		if err != nil {
			initErr = fmt.Errorf("failed to create resource: %w", err)
			return
		}

		// Create a PeriodicReader for pushing metrics at intervals.
		readerOpts := []sdkmetric.PeriodicReaderOption{sdkmetric.WithInterval(cfg.PushInterval)}
		pr := sdkmetric.NewPeriodicReader(exporter, readerOpts...)

		// Build custom histogram views if provided.
		customViews := buildCustomViews(cfg.CustomHistogramViews)

		// Build MeterProvider with optional custom views.
		mp := sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(pr),
			sdkmetric.WithResource(r),
			sdkmetric.WithView(customViews...),
		)

		// Register the global MeterProvider.
		meterProvider = mp
		apimetric.SetMeterProvider(meterProvider)

		// Define a shutdown function.
		shutdownFunc = func(shutdownCtx context.Context) error {
			// flush & stop
			return mp.Shutdown(shutdownCtx)
		}

		// Mark as initialized.
		mu.Lock()
		initialized = true
		mu.Unlock()

		log.Printf("[metrics] OTLP metrics initialized. Endpoint=%s Insecure=%v", cfg.OTLPEndpoint, cfg.OTLPInsecure)
	})
	return initErr
}

// NewConfig creates a new Config with the provided options.
func NewConfig(endpoint, serviceName, environment string, opts ...Option) Config {
	c := &Config{
		OTLPEndpoint:         endpoint,
		OTLPInsecure:         true,
		OTLPCAFile:           "",
		PushInterval:         10 * time.Second,
		ServiceName:          serviceName,
		Environment:          environment,
		CustomHistogramViews: nil,
	}

	// Apply all the user-supplied options.
	for _, opt := range opts {
		opt(c)
	}

	return *c
}

// buildCustomViews creates a slice of custom views from the provided config.
func buildCustomViews(histogramViews []InstrumentViewConfig) []sdkmetric.View {
	var views []sdkmetric.View

	for _, v := range histogramViews {
		// Create a new view with explicit bucket boundaries.
		v := sdkmetric.NewView(
			sdkmetric.Instrument{
				Name: v.InstrumentName,
			},
			sdkmetric.Stream{
				Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
					Boundaries: v.Buckets,
				},
			},
		)
		views = append(views, v)
	}

	return views
}

// createOTLPExporter creates an OTLP gRPC exporter with the provided config.
func createOTLPExporter(ctx context.Context, cfg Config) (sdkmetric.Exporter, error) {
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint),
	}

	// Set up secure or insecure connection.
	if cfg.OTLPInsecure {
		opts = append(opts, otlpmetricgrpc.WithTLSCredentials(insecure.NewCredentials()))
	} else {
		creds, err := credentials.NewClientTLSFromFile(cfg.OTLPCAFile, "")
		if err != nil {
			return nil, fmt.Errorf("failed to load CA file: %w", err)
		}
		opts = append(opts, otlpmetricgrpc.WithTLSCredentials(creds))
	}
	return otlpmetricgrpc.New(ctx, opts...)
}

// ShutdownMetrics flushes and stops the global MeterProvider.
func ShutdownMetrics(ctx context.Context) error {
	var err error
	shutdownOnce.Do(func() {
		mu.RLock()
		// If not initialized, return early
		if !initialized {
			mu.RUnlock()
			return
		}
		mu.RUnlock()

		// Check if environment variable is set to skip flushing, this is intended for testing only.
		if os.Getenv("METRICS_SKIP_FLUSH") == "1" {
			log.Printf("[metrics] METRICS_SKIP_FLUSH is set; skipping flush in ShutdownMetrics")
			err = nil
		} else if shutdownFunc != nil {
			err = shutdownFunc(ctx)
			if err != nil {
				log.Printf("[metrics] Shutdown error: %v", err)
			}
		}

		// Mark as uninitialized.
		mu.Lock()
		initialized = false
		mu.Unlock()
	})
	return err
}

// GetMeter returns a Meter from the global provider or a no-op if uninitialized.
func GetMeter(name string) metric.Meter {
	mu.RLock()
	defer mu.RUnlock()

	if !initialized || meterProvider == nil {
		return apimetric.GetMeterProvider().Meter(name)
	}
	return meterProvider.Meter(name)
}

// validateConfig ensures that mandatory fields in the Config are set,
// and returns an error if the configuration is invalid.
func validateConfig(cfg Config) error {
	if cfg.OTLPEndpoint == "" {
		return errors.New("OTLPEndpoint is required (e.g. 'localhost:4317')")
	}
	if cfg.ServiceName == "" {
		return errors.New("ServiceName is required")
	}
	if cfg.Environment == "" {
		return errors.New("Environment is required (e.g. 'dev', 'staging', 'prod')")
	}
	if cfg.PushInterval <= 0 {
		return errors.New("PushInterval must be greater than 0")
	}
	if !cfg.OTLPInsecure && cfg.OTLPCAFile == "" {
		return errors.New("CA file required for secure mode")
	}

	// Validate custom histogram views.
	for _, hv := range cfg.CustomHistogramViews {
		if hv.InstrumentName == "" {
			return fmt.Errorf("found a CustomHistogramView with empty InstrumentName")
		}
		if len(hv.Buckets) == 0 || len(hv.Buckets) < 2 {
			return fmt.Errorf("found a CustomHistogramView with less than 2 Buckets")
		}
	}

	return nil
}
