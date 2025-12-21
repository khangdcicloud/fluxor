package web

import (
	"testing"
)

func TestCCUBasedConfig(t *testing.T) {
	config := CCUBasedConfig(":8080", 5000, 500)
	
	// Verify configuration
	if config.MaxQueue+config.Workers < 5000 {
		t.Errorf("Total capacity (%d) should be at least 5000", config.MaxQueue+config.Workers)
	}
	
	if config.Workers < 50 {
		t.Error("Workers should be at least 50")
	}
	
	if config.MaxQueue < 100 {
		t.Error("MaxQueue should be at least 100")
	}
}

func TestCCUBasedConfigWithUtilization(t *testing.T) {
	maxCCU := 5000
	utilizationPercent := 60
	config := CCUBasedConfigWithUtilization(":8080", maxCCU, utilizationPercent)
	
	// Calculate expected normal capacity (60% of max)
	expectedNormalCapacity := int(float64(maxCCU) * float64(utilizationPercent) / 100.0)
	actualNormalCapacity := config.MaxQueue + config.Workers
	
	// Allow some tolerance for rounding
	tolerance := 50
	if actualNormalCapacity < expectedNormalCapacity-tolerance || actualNormalCapacity > expectedNormalCapacity+tolerance {
		t.Errorf("Normal capacity = %d, want ~%d (60%% of %d)", actualNormalCapacity, expectedNormalCapacity, maxCCU)
	}
	
	if config.Workers < 50 {
		t.Error("Workers should be at least 50")
	}
	
	if config.MaxQueue < 100 {
		t.Error("MaxQueue should be at least 100")
	}
	
	// MaxConns should allow up to maxCCU
	if config.MaxConns < maxCCU {
		t.Errorf("MaxConns = %d, should be at least %d", config.MaxConns, maxCCU)
	}
}

func TestBackpressureController(t *testing.T) {
	normalCapacity := 3000 // 60% of 5000 max
	bc := NewBackpressureController(normalCapacity, 60)
	
	// Test capacity acquisition (up to normal capacity)
	for i := 0; i < normalCapacity; i++ {
		if !bc.TryAcquire() {
			t.Errorf("Should acquire capacity for request %d", i)
		}
	}
	
	// Test overflow rejection (fail-fast) - beyond normal capacity
	if bc.TryAcquire() {
		t.Error("Should reject request when normal capacity exceeded")
	}
	
	// Test metrics
	metrics := bc.GetMetrics()
	if metrics.CurrentLoad != int64(normalCapacity) {
		t.Errorf("CurrentLoad = %d, want %d", metrics.CurrentLoad, normalCapacity)
	}
	if metrics.NormalCapacity != int64(normalCapacity) {
		t.Errorf("NormalCapacity = %d, want %d", metrics.NormalCapacity, normalCapacity)
	}
	if metrics.RejectedCount == 0 {
		t.Error("Should have rejected at least one request")
	}
	if metrics.Utilization < 100.0 {
		t.Errorf("Utilization should be >= 100%% when at capacity, got %.2f%%", metrics.Utilization)
	}
	
	// Test release
	bc.Release()
	metrics = bc.GetMetrics()
	if metrics.CurrentLoad != int64(normalCapacity-1) {
		t.Errorf("CurrentLoad = %d, want %d", metrics.CurrentLoad, normalCapacity-1)
	}
	
	// After release, should be able to acquire again
	if !bc.TryAcquire() {
		t.Error("Should acquire capacity after release")
	}
}
