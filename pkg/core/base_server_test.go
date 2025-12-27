package core

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestBaseServer_Start_FailFast_RollbackStartedOnHookError(t *testing.T) {
	t.Parallel()

	gocmd := NewGoCMD(context.Background())
	bs := NewBaseServer("test", gocmd)
	bs.SetHooks(
		func() error { return errors.New("boom") },
		func() error { return nil },
	)

	if err := bs.Start(); err == nil {
		t.Fatalf("expected error")
	}
	if bs.IsStarted() {
		t.Fatalf("expected started=false after start hook error")
	}
}

func TestBaseServer_Start_BlocksButMarksStarted(t *testing.T) {
	gocmd := NewGoCMD(context.Background())
	bs := NewBaseServer("test", gocmd)

	release := make(chan struct{})
	var entered int64
	bs.SetHooks(
		func() error {
			atomic.AddInt64(&entered, 1)
			<-release
			return nil
		},
		func() error { return nil },
	)

	errCh := make(chan error, 1)
	go func() { errCh <- bs.Start() }()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt64(&entered) == 1 && bs.IsStarted() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if atomic.LoadInt64(&entered) != 1 {
		close(release)
		t.Fatalf("expected start hook to be entered")
	}
	if !bs.IsStarted() {
		close(release)
		t.Fatalf("expected IsStarted()=true while start hook is blocking")
	}

	close(release)

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("unexpected start error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("start did not return after unblocking hook")
	}
}
