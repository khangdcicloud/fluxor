package core

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestDeploymentState_SyncVerticle tests state transitions for synchronous verticles
func TestDeploymentState_SyncVerticle(t *testing.T) {
	ctx := context.Background()
	vx := NewGoCMD(ctx).(*gocmd)
	defer vx.Close()

	verticle := &testVerticle{}
	deploymentID, err := vx.DeployVerticle(verticle)
	if err != nil {
		t.Fatalf("DeployVerticle() error = %v", err)
	}

	// Wait for async start to complete
	// Since Start() is async, we need to wait for it to finish
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		vx.mu.RLock()
		dep, exists := vx.deployments[deploymentID]
		if exists && dep.state == DeploymentStateStarted {
			vx.mu.RUnlock()
			// State is STARTED - success!
			return
		}
		vx.mu.RUnlock()
		time.Sleep(10 * time.Millisecond)
	}

	// After successful deploy, state should be STARTED
	vx.mu.RLock()
	dep, exists := vx.deployments[deploymentID]
	vx.mu.RUnlock()

	if !exists {
		t.Fatalf("deployment %s should exist", deploymentID)
	}
	if dep.state != DeploymentStateStarted {
		t.Errorf("expected state STARTED, got %d", dep.state)
	}
}

// TestDeploymentState_SyncVerticleFailure tests state when Start() fails
func TestDeploymentState_SyncVerticleFailure(t *testing.T) {
	ctx := context.Background()
	vx := NewGoCMD(ctx).(*gocmd)
	defer vx.Close()

	verticle := &failingStartVerticle{}
	deploymentID, err := vx.DeployVerticle(verticle)
	if err != nil {
		t.Fatalf("DeployVerticle() should not return error (start is async), got %v", err)
	}

	// Wait for async start to complete and fail
	// The goroutine will remove the deployment from map on failure
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		vx.mu.RLock()
		_, exists := vx.deployments[deploymentID]
		vx.mu.RUnlock()
		if !exists {
			// Deployment removed - failure handled correctly
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Deployment should be removed from map on failure
	vx.mu.RLock()
	count := len(vx.deployments)
	vx.mu.RUnlock()

	if count != 0 {
		t.Errorf("expected 0 deployments after failure, got %d", count)
	}
}

// asyncTestVerticle is an async verticle for testing state transitions
type asyncTestVerticle struct {
	startCalled int32
	stopCalled  int32
	startDelay  time.Duration
	startErr    error
	stopErr     error
	startDone   chan struct{}
	stopDone    chan struct{}
}

func newAsyncTestVerticle(startDelay time.Duration) *asyncTestVerticle {
	return &asyncTestVerticle{
		startDelay: startDelay,
		startDone:  make(chan struct{}),
		stopDone:   make(chan struct{}),
	}
}

func (v *asyncTestVerticle) Start(ctx FluxorContext) error {
	return nil
}

func (v *asyncTestVerticle) Stop(ctx FluxorContext) error {
	return nil
}

func (v *asyncTestVerticle) AsyncStart(ctx FluxorContext, resultHandler func(error)) {
	atomic.AddInt32(&v.startCalled, 1)
	go func() {
		if v.startDelay > 0 {
			time.Sleep(v.startDelay)
		}
		resultHandler(v.startErr)
		close(v.startDone)
	}()
}

func (v *asyncTestVerticle) AsyncStop(ctx FluxorContext, resultHandler func(error)) {
	atomic.AddInt32(&v.stopCalled, 1)
	go func() {
		resultHandler(v.stopErr)
		close(v.stopDone)
	}()
}

// TestDeploymentState_AsyncVerticle_Pending tests that async verticle starts in PENDING state
func TestDeploymentState_AsyncVerticle_Pending(t *testing.T) {
	ctx := context.Background()
	vx := NewGoCMD(ctx).(*gocmd)
	defer vx.Close()

	// Use a longer delay to observe PENDING state
	verticle := newAsyncTestVerticle(100 * time.Millisecond)

	deploymentID, err := vx.DeployVerticle(verticle)
	if err != nil {
		t.Fatalf("DeployVerticle() error = %v", err)
	}

	// Immediately after deploy, state should be PENDING
	vx.mu.RLock()
	dep, exists := vx.deployments[deploymentID]
	state := dep.state
	vx.mu.RUnlock()

	if !exists {
		t.Fatalf("deployment %s should exist", deploymentID)
	}
	if state != DeploymentStatePending {
		t.Errorf("expected state PENDING immediately after deploy, got %d", state)
	}

	// Wait for async start to complete
	select {
	case <-verticle.startDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("async start timed out")
	}

	// After completion, state should be STARTED
	vx.mu.RLock()
	state = dep.state
	vx.mu.RUnlock()

	if state != DeploymentStateStarted {
		t.Errorf("expected state STARTED after async complete, got %d", state)
	}
}

// TestDeploymentState_AsyncVerticle_Failed tests state when AsyncStart fails
func TestDeploymentState_AsyncVerticle_Failed(t *testing.T) {
	ctx := context.Background()
	vx := NewGoCMD(ctx).(*gocmd)
	defer vx.Close()

	verticle := newAsyncTestVerticle(0)
	verticle.startErr = errors.New("async start failed")

	deploymentID, err := vx.DeployVerticle(verticle)
	if err != nil {
		t.Fatalf("DeployVerticle() should not return error for async verticle, got %v", err)
	}

	// Wait for async start callback
	select {
	case <-verticle.startDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("async start timed out")
	}

	// Small delay to let callback complete
	time.Sleep(10 * time.Millisecond)

	// After failure, deployment should be removed from map
	vx.mu.RLock()
	_, exists := vx.deployments[deploymentID]
	vx.mu.RUnlock()

	if exists {
		t.Errorf("deployment should be removed after async start failure")
	}
}

// TestDeploymentState_Undeploy_Stopping tests state transitions during undeploy
func TestDeploymentState_Undeploy_Stopping(t *testing.T) {
	ctx := context.Background()
	vx := NewGoCMD(ctx).(*gocmd)
	defer vx.Close()

	verticle := &testVerticle{}
	deploymentID, err := vx.DeployVerticle(verticle)
	if err != nil {
		t.Fatalf("DeployVerticle() error = %v", err)
	}

	// Wait for async start to complete before undeploying
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		vx.mu.RLock()
		dep, exists := vx.deployments[deploymentID]
		if exists && dep.state == DeploymentStateStarted {
			vx.mu.RUnlock()
			break
		}
		vx.mu.RUnlock()
		time.Sleep(10 * time.Millisecond)
	}

	// Undeploy should transition to STOPPED
	err = vx.UndeployVerticle(deploymentID)
	if err != nil {
		t.Errorf("UndeployVerticle() error = %v", err)
	}

	// Deployment should be removed from map
	vx.mu.RLock()
	_, exists := vx.deployments[deploymentID]
	vx.mu.RUnlock()

	if exists {
		t.Errorf("deployment should be removed after undeploy")
	}
}

// TestDeploymentState_Undeploy_PendingRejected tests that pending deployments cannot be undeployed
func TestDeploymentState_Undeploy_PendingRejected(t *testing.T) {
	ctx := context.Background()
	vx := NewGoCMD(ctx).(*gocmd)
	defer vx.Close()

	// Use a longer delay to keep verticle in PENDING state
	verticle := newAsyncTestVerticle(500 * time.Millisecond)

	deploymentID, err := vx.DeployVerticle(verticle)
	if err != nil {
		t.Fatalf("DeployVerticle() error = %v", err)
	}

	// Try to undeploy while PENDING
	err = vx.UndeployVerticle(deploymentID)
	if err == nil {
		t.Errorf("UndeployVerticle() should fail for PENDING deployment")
	}

	// Check error type
	fluxorErr, ok := err.(*EventBusError)
	if !ok {
		t.Errorf("expected *Error, got %T", err)
	} else if fluxorErr.Code != "DEPLOYMENT_PENDING" {
		t.Errorf("expected code DEPLOYMENT_PENDING, got %s", fluxorErr.Code)
	}
}

// TestDeploymentState_Undeploy_DoubleUndeploy tests that double undeploy is rejected
func TestDeploymentState_Undeploy_DoubleUndeploy(t *testing.T) {
	ctx := context.Background()
	vx := NewGoCMD(ctx).(*gocmd)
	defer vx.Close()

	verticle := &testVerticle{}
	deploymentID, err := vx.DeployVerticle(verticle)
	if err != nil {
		t.Fatalf("DeployVerticle() error = %v", err)
	}

	// Wait for async start to complete before undeploying
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		vx.mu.RLock()
		dep, exists := vx.deployments[deploymentID]
		if exists && dep.state == DeploymentStateStarted {
			vx.mu.RUnlock()
			break
		}
		vx.mu.RUnlock()
		time.Sleep(10 * time.Millisecond)
	}

	// First undeploy should succeed
	err = vx.UndeployVerticle(deploymentID)
	if err != nil {
		t.Errorf("first UndeployVerticle() error = %v", err)
	}

	// Second undeploy should fail (deployment not found)
	err = vx.UndeployVerticle(deploymentID)
	if err == nil {
		t.Errorf("second UndeployVerticle() should fail")
	}
}

// TestDeploymentState_AsyncUndeploy tests async verticle undeploy state transitions
func TestDeploymentState_AsyncUndeploy(t *testing.T) {
	ctx := context.Background()
	vx := NewGoCMD(ctx).(*gocmd)
	defer vx.Close()

	verticle := newAsyncTestVerticle(0)

	deploymentID, err := vx.DeployVerticle(verticle)
	if err != nil {
		t.Fatalf("DeployVerticle() error = %v", err)
	}

	// Wait for async start
	select {
	case <-verticle.startDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("async start timed out")
	}
	time.Sleep(10 * time.Millisecond)

	// Undeploy
	err = vx.UndeployVerticle(deploymentID)
	if err != nil {
		t.Errorf("UndeployVerticle() error = %v", err)
	}

	// Wait for async stop
	select {
	case <-verticle.stopDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("async stop timed out")
	}

	// Verify stop was called
	if atomic.LoadInt32(&verticle.stopCalled) != 1 {
		t.Errorf("AsyncStop should be called once")
	}
}

// TestDeploymentState_ConcurrentDeploy tests concurrent deployments
func TestDeploymentState_ConcurrentDeploy(t *testing.T) {
	ctx := context.Background()
	vx := NewGoCMD(ctx).(*gocmd)
	defer vx.Close()

	const numDeployments = 10
	var wg sync.WaitGroup
	deploymentIDs := make(chan string, numDeployments)
	errs := make(chan error, numDeployments)

	for i := 0; i < numDeployments; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			verticle := &testVerticle{}
			id, err := vx.DeployVerticle(verticle)
			if err != nil {
				errs <- err
				return
			}
			deploymentIDs <- id
		}()
	}

	wg.Wait()
	close(deploymentIDs)
	close(errs)

	// Check for errors
	for err := range errs {
		t.Errorf("concurrent deploy error: %v", err)
	}

	// Collect all deployment IDs
	ids := make([]string, 0, numDeployments)
	for id := range deploymentIDs {
		ids = append(ids, id)
	}

	// Wait for all async starts to complete
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		allStarted := true
		for _, id := range ids {
			vx.mu.RLock()
			dep, exists := vx.deployments[id]
			vx.mu.RUnlock()
			if !exists || dep.state != DeploymentStateStarted {
				allStarted = false
				break
			}
		}
		if allStarted {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Verify all deployments exist and are STARTED
	count := 0
	for _, id := range ids {
		count++
		vx.mu.RLock()
		dep, exists := vx.deployments[id]
		vx.mu.RUnlock()

		if !exists {
			t.Errorf("deployment %s should exist", id)
			continue
		}
		if dep.state != DeploymentStateStarted {
			t.Errorf("deployment %s should be STARTED, got %d", id, dep.state)
		}
	}

	if count != numDeployments {
		t.Errorf("expected %d deployments, got %d", numDeployments, count)
	}

	// Verify count
	if vx.DeploymentCount() != numDeployments {
		t.Errorf("DeploymentCount() = %d, want %d", vx.DeploymentCount(), numDeployments)
	}
}

// TestDeploymentState_Constants tests that DeploymentState constants have expected values
func TestDeploymentState_Constants(t *testing.T) {
	// Verify iota ordering
	if DeploymentStatePending != 0 {
		t.Errorf("DeploymentStatePending should be 0, got %d", DeploymentStatePending)
	}
	if DeploymentStateStarted != 1 {
		t.Errorf("DeploymentStateStarted should be 1, got %d", DeploymentStateStarted)
	}
	if DeploymentStateFailed != 2 {
		t.Errorf("DeploymentStateFailed should be 2, got %d", DeploymentStateFailed)
	}
	if DeploymentStateStopping != 3 {
		t.Errorf("DeploymentStateStopping should be 3, got %d", DeploymentStateStopping)
	}
	if DeploymentStateStopped != 4 {
		t.Errorf("DeploymentStateStopped should be 4, got %d", DeploymentStateStopped)
	}
}

// TestDeploymentState_CloseUndeploysAll tests that Close() undeploys all verticles
func TestDeploymentState_CloseUndeploysAll(t *testing.T) {
	ctx := context.Background()
	vx := NewGoCMD(ctx).(*gocmd)

	verticles := make([]*testVerticle, 5)
	for i := 0; i < 5; i++ {
		verticles[i] = &testVerticle{}
		_, err := vx.DeployVerticle(verticles[i])
		if err != nil {
			t.Fatalf("DeployVerticle() error = %v", err)
		}
	}

	// Wait for all async starts to complete
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		allStarted := true
		for _, v := range verticles {
			if !v.started {
				allStarted = false
				break
			}
		}
		if allStarted {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Verify all started
	for i, v := range verticles {
		if !v.started {
			t.Errorf("verticle %d should be started", i)
		}
	}

	// Close should undeploy all
	err := vx.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify all stopped
	for i, v := range verticles {
		if !v.stopped {
			t.Errorf("verticle %d should be stopped after Close()", i)
		}
	}

	// Verify no deployments remain
	vx.mu.RLock()
	count := len(vx.deployments)
	vx.mu.RUnlock()

	if count != 0 {
		t.Errorf("expected 0 deployments after Close(), got %d", count)
	}
}
