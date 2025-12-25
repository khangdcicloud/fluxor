package verticles

import (
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/examples/load-balancing/contracts"
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
)

// MasterVerticle acts as a load balancer and gateway
type MasterVerticle struct {
	*core.BaseVerticle
	
	workerIDs []string
	counter   uint64
	
	httpServer *web.FastHTTPServer
	tcpListener net.Listener
	logger      core.Logger
}

func NewMasterVerticle(workerIDs []string) *MasterVerticle {
	return &MasterVerticle{
		BaseVerticle: core.NewBaseVerticle("master"),
		workerIDs:    workerIDs,
		logger:       core.NewDefaultLogger(),
	}
}

func (v *MasterVerticle) doStart(ctx core.FluxorContext) error {
	v.logger.Info("Master starting...")

	// 1. Start HTTP Server
	v.startHTTPServer(ctx)

	// 2. Start TCP Server
	go v.startTCPServer(ctx)

	return nil
}

func (v *MasterVerticle) doStop(ctx core.FluxorContext) error {
	if v.httpServer != nil {
		_ = v.httpServer.Stop()
	}
	if v.tcpListener != nil {
		_ = v.tcpListener.Close()
	}
	return nil
}

func (v *MasterVerticle) startHTTPServer(ctx core.FluxorContext) {
	cfg := web.DefaultFastHTTPServerConfig(":8080")
	v.httpServer = web.NewFastHTTPServer(ctx.Vertx(), cfg)
	
	r := v.httpServer.FastRouter()
	r.GETFast("/process", func(c *web.FastRequestContext) error {
		payload := c.Query("data")
		if payload == "" {
			payload = "default-data"
		}

		// Load Balance
		workerAddr := v.nextWorkerAddress()
		
		req := contracts.WorkRequest{
			ID:      fmt.Sprintf("http-%d", time.Now().UnixNano()),
			Payload: payload,
		}

		// Call Worker
		reply, err := c.EventBus.Request(workerAddr, req, 5*time.Second)
		if err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}

		var resp contracts.WorkResponse
		_ = reply.DecodeBody(&resp)
		
		return c.JSON(200, resp)
	})

	go func() {
		if err := v.httpServer.Start(); err != nil {
			v.logger.Errorf("HTTP Server failed: %v", err)
		}
	}()
	v.logger.Info("HTTP Server listening on :8080")
}

func (v *MasterVerticle) startTCPServer(ctx core.FluxorContext) {
	addr := ":9090"
	l, err := net.Listen("tcp", addr)
	if err != nil {
		v.logger.Errorf("TCP listen failed: %v", err)
		return
	}
	v.tcpListener = l
	v.logger.Infof("TCP Server listening on %s", addr)

	for {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		go v.handleTCPConnection(ctx, conn)
	}
}

func (v *MasterVerticle) handleTCPConnection(ctx core.FluxorContext, conn net.Conn) {
	defer conn.Close()
	
	// Simple protocol: Read line -> Process -> Write line
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}
	
	payload := string(buf[:n])
	
	// Load Balance
	workerAddr := v.nextWorkerAddress()
	
	req := contracts.WorkRequest{
		ID:      fmt.Sprintf("tcp-%d", time.Now().UnixNano()),
		Payload: payload,
	}

	// Call Worker via EventBus
	reply, err := v.EventBus().Request(workerAddr, req, 5*time.Second)
	if err != nil {
		conn.Write([]byte(fmt.Sprintf("Error: %v\n", err)))
		return
	}

	var resp contracts.WorkResponse
	_ = reply.DecodeBody(&resp)
	
	conn.Write([]byte(fmt.Sprintf("Processed by %s: %s\n", resp.Worker, resp.Result)))
}

func (v *MasterVerticle) nextWorkerAddress() string {
	idx := atomic.AddUint64(&v.counter, 1) % uint64(len(v.workerIDs))
	workerID := v.workerIDs[idx]
	return fmt.Sprintf("%s.%s", contracts.WorkerAddress, workerID)
}
