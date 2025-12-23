package web

import (
	"sync/atomic"
	"time"
)

// BackpressureController manages backpressure for the server
// Ensures system stability under high load by rejecting overflow requests
// Normal capacity is set to target utilization (e.g., 67% of max capacity)
type BackpressureController struct {
	normalCapacity int64 // Normal capacity (target utilization, e.g., 67% of max)
	currentLoad    int64 // Current load (atomic)
	rejectedCount  int64 // Rejected requests count
	lastReset      int64 // Last reset time (unix timestamp)
	resetInterval  int64 // Reset interval in seconds
}

// NewBackpressureController creates a new backpressure controller
// normalCapacity: Target capacity for normal operations (e.g., 67% of max)
// This ensures system operates at target utilization under normal load
func NewBackpressureController(normalCapacity int, resetIntervalSeconds int64) *BackpressureController {
	return &BackpressureController{
		normalCapacity: int64(normalCapacity),
		currentLoad:    0,
		rejectedCount:  0,
		lastReset:      time.Now().Unix(),
		resetInterval:  resetIntervalSeconds,
	}
}

// TryAcquire attempts to acquire capacity (fail-fast)
// Returns true if normal capacity available, false if should reject (503)
// Normal capacity = target utilization (e.g., 67% of max capacity)
func (bc *BackpressureController) TryAcquire() bool {
	// Reset counters periodically
	now := time.Now().Unix()
	if now-bc.lastReset > bc.resetInterval {
		atomic.StoreInt64(&bc.currentLoad, 0)
		atomic.StoreInt64(&bc.lastReset, now)
	}

	// Check current load against normal capacity (target utilization)
	current := atomic.LoadInt64(&bc.currentLoad)
	if current >= bc.normalCapacity {
		// Fail-fast: normal capacity exceeded, reject immediately
		// This maintains target utilization (e.g., 67%) under normal conditions
		atomic.AddInt64(&bc.rejectedCount, 1)
		return false
	}

	// Acquire capacity
	atomic.AddInt64(&bc.currentLoad, 1)
	return true
}

// Release releases capacity
func (bc *BackpressureController) Release() {
	atomic.AddInt64(&bc.currentLoad, -1)
}

// GetMetrics returns current backpressure metrics
func (bc *BackpressureController) GetMetrics() BackpressureMetrics {
	currentLoad := atomic.LoadInt64(&bc.currentLoad)
	return BackpressureMetrics{
		NormalCapacity: bc.normalCapacity,
		CurrentLoad:    currentLoad,
		RejectedCount:  atomic.LoadInt64(&bc.rejectedCount),
		Utilization:    float64(currentLoad) / float64(bc.normalCapacity) * 100,
	}
}

// BackpressureMetrics provides backpressure statistics
type BackpressureMetrics struct {
	NormalCapacity int64   // Normal capacity (target utilization)
	CurrentLoad    int64   // Current load
	RejectedCount  int64   // Total rejected requests
	Utilization    float64 // Utilization percentage (relative to normal capacity)
}
