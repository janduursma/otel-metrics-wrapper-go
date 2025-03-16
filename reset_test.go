//go:build test
// +build test

package metrics

import "sync"

// ResetState resets the package-level state (including initOnce, initialized flag,
// meterProvider, and shutdownFunc) so that tests can reinitialize the metrics pipeline.
// This function is intended for testing only.
func ResetState() {
	// Reset the sync.Once so that InitMetrics will run again.
	initOnce = sync.Once{}
	initialized = false
	meterProvider = nil
	shutdownOnce = sync.Once{}
	shutdownFunc = nil
}
