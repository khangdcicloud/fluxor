package tcp

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

func TestNewTCPServer_FailFast_NilVertxPanics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic for nil vertx")
		}
	}()
	_ = NewTCPServer(nil, DefaultTCPServerConfig(":0"))
}

func TestTCPServer_SetHandler_FailFast_NilPanics(t *testing.T) {
	t.Parallel()
	vertx := core.NewVertx(context.Background())
	s := NewTCPServer(vertx, DefaultTCPServerConfig("127.0.0.1:0"))
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic for nil handler")
		}
	}()
	s.SetHandler(nil)
}

func TestTCPServer_StartStop_HandlesConnection(t *testing.T) {
	vertx := core.NewVertx(context.Background())
	cfg := DefaultTCPServerConfig("127.0.0.1:0")
	cfg.Workers = 2
	cfg.MaxQueue = 10
	cfg.ReadTimeout = 250 * time.Millisecond
	cfg.WriteTimeout = 250 * time.Millisecond

	s := NewTCPServer(vertx, cfg)

	var handled int64
	s.SetHandler(func(ctx *ConnContext) error {
		buf := make([]byte, 1)
		_, _ = ctx.Conn.Read(buf) // best-effort
		atomic.AddInt64(&handled, 1)
		return nil
	})

	startErrCh := make(chan error, 1)
	go func() {
		startErrCh <- s.Start()
	}()

	// Wait until listener is up and we have an actual port.
	var addr string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		addr = s.ListeningAddr()
		if addr != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if addr == "" {
		_ = s.Stop()
		t.Fatalf("server did not start listening in time")
	}

	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		_ = s.Stop()
		t.Fatalf("dial failed: %v", err)
	}
	_, _ = conn.Write([]byte{0x01})
	_ = conn.Close()

	// Wait for handler to run.
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt64(&handled) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if atomic.LoadInt64(&handled) == 0 {
		_ = s.Stop()
		t.Fatalf("expected handler to be called")
	}

	if err := s.Stop(); err != nil {
		t.Fatalf("stop failed: %v", err)
	}

	select {
	case err := <-startErrCh:
		if err != nil {
			t.Fatalf("start returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("start did not exit after stop")
	}
}
