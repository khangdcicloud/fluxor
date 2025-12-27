package verticles

import (
	"bufio"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/examples/load-balancing/contracts"
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/tcp"
	"github.com/fluxorio/fluxor/pkg/web"
)

// MasterVerticle acts as a load balancer and gateway
type MasterVerticle struct {
	*core.BaseVerticle

	workerIDs []string
	counter   uint64

	httpPort     string
	tcpAddr      string
	httpVerticle *web.HttpVerticle
	tcpServer    *tcp.TCPServer
	logger       core.Logger
}

func NewMasterVerticle(workerIDs []string) *MasterVerticle {
	return &MasterVerticle{
		BaseVerticle: core.NewBaseVerticle("master"),
		workerIDs:    workerIDs,
		logger:       core.NewDefaultLogger(),
	}
}

// Start overrides BaseVerticle.Start to call doStart
func (v *MasterVerticle) Start(ctx core.FluxorContext) error {
	if err := v.BaseVerticle.Start(ctx); err != nil {
		return err
	}
	return v.doStart(ctx)
}

func (v *MasterVerticle) doStart(ctx core.FluxorContext) error {
	v.logger.Info("Master starting...")

	// Config (optional)
	cfg := ctx.Config()
	if p, ok := cfg["http_port"].(string); ok && p != "" {
		v.httpPort = p
	}
	if v.httpPort == "" {
		v.httpPort = "8080"
	}
	if a, ok := cfg["tcp_addr"].(string); ok && a != "" {
		v.tcpAddr = a
	}
	if v.tcpAddr == "" {
		v.tcpAddr = ":9090"
	}

	// 1. Start HTTP gateway (HttpVerticle)
	if err := v.startHTTPVerticle(ctx); err != nil {
		return err
	}

	// 2. Start TCP gateway (pkg/tcp server)
	v.startTCPServer(ctx)

	return nil
}

func (v *MasterVerticle) doStop(ctx core.FluxorContext) error {
	if v.httpVerticle != nil {
		_ = v.httpVerticle.Stop(ctx)
	}
	if v.tcpServer != nil {
		_ = v.tcpServer.Stop()
	}
	return nil
}

func (v *MasterVerticle) startHTTPVerticle(ctx core.FluxorContext) error {
	r := web.NewRouter()

	r.GET("/process", func(c *web.RequestContext) error {
		payload := ""
		if c.Request != nil {
			payload = c.Request.URL.Query().Get("data")
		}
		if payload == "" {
			payload = "default-data"
		}

		workerAddr := v.nextWorkerAddress()
		req := contracts.WorkRequest{
			ID:      fmt.Sprintf("http-%d", time.Now().UnixNano()),
			Payload: payload,
		}

		reply, err := v.EventBus().Request(workerAddr, req, 5*time.Second)
		if err != nil {
			return c.JSON(502, map[string]any{"error": err.Error()})
		}

		var resp contracts.WorkResponse
		_ = reply.DecodeBody(&resp)
		return c.JSON(200, resp)
	})

	r.GET("/status", func(c *web.RequestContext) error {
		type status struct {
			Role     string            `json:"role"`
			Workers  []string          `json:"workers"`
			TCPAddr  string            `json:"tcp_addr"`
			HTTPPort string            `json:"http_port"`
			Metrics  tcp.ServerMetrics `json:"tcp_metrics"`
		}

		m := tcp.ServerMetrics{}
		if v.tcpServer != nil {
			m = v.tcpServer.Metrics()
		}
		return c.JSON(200, status{
			Role:     "master",
			Workers:  v.workerIDs,
			TCPAddr:  v.tcpAddr,
			HTTPPort: v.httpPort,
			Metrics:  m,
		})
	})

	v.httpVerticle = web.NewHttpVerticle(v.httpPort, r)
	v.logger.Info(fmt.Sprintf("HTTP Server listening on :%s", v.httpPort))
	return v.httpVerticle.Start(ctx)
}

func (v *MasterVerticle) startTCPServer(ctx core.FluxorContext) {
	cfg := tcp.DefaultTCPServerConfig(v.tcpAddr)
	v.tcpServer = tcp.NewTCPServer(ctx.GoCMD(), cfg)

	v.tcpServer.SetHandler(func(c *tcp.ConnContext) error {
		// Simple protocol: one line in, one line out.
		rd := bufio.NewReader(c.Conn)
		line, err := rd.ReadBytes('\n')
		if err != nil && len(line) == 0 {
			return err
		}

		// Trim common line endings/spaces.
		payload := string(line)
		for len(payload) > 0 && (payload[len(payload)-1] == '\n' || payload[len(payload)-1] == '\r') {
			payload = payload[:len(payload)-1]
		}

		workerAddr := v.nextWorkerAddress()
		req := contracts.WorkRequest{
			ID:      fmt.Sprintf("tcp-%d", time.Now().UnixNano()),
			Payload: payload,
		}

		reply, reqErr := c.EventBus.Request(workerAddr, req, 5*time.Second)
		if reqErr != nil {
			_, _ = c.Conn.Write([]byte(fmt.Sprintf("Error: %v\n", reqErr)))
			return nil
		}

		var resp contracts.WorkResponse
		_ = reply.DecodeBody(&resp)
		out := map[string]any{
			"id":     resp.ID,
			"result": resp.Result,
			"worker": resp.Worker,
		}
		b, _ := json.Marshal(out)
		_, _ = c.Conn.Write(append(b, '\n'))
		return nil
	})

	go func() {
		v.logger.Info(fmt.Sprintf("TCP Server listening on %s", v.tcpAddr))
		if err := v.tcpServer.Start(); err != nil {
			v.logger.Error(fmt.Sprintf("TCP Server failed: %v", err))
		}
	}()
}

func (v *MasterVerticle) nextWorkerAddress() string {
	idx := atomic.AddUint64(&v.counter, 1) % uint64(len(v.workerIDs))
	workerID := v.workerIDs[idx]
	return fmt.Sprintf("%s.%s", contracts.WorkerAddress, workerID)
}
