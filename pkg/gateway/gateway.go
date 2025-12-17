package gateway

import (
	"context"
	"fmt"
	"log"

	"github.com/fluxor-io/fluxor/pkg/bus"
	"github.com/fluxor-io/fluxor/pkg/httpx"
	"github.com/fluxor-io/fluxor/pkg/reactor"
)

type Gateway struct {
	server *httpx.Server
	port   int
}

func NewGateway(port int, reactor *reactor.Reactor, bus bus.Bus) *Gateway {
	g := &Gateway{
		port: port,
	}

	// Use httpx.NewServer from our shared package
	g.server = httpx.NewServer(port, reactor, bus)

	return g
}

func (g *Gateway) Start(ctx context.Context) error {
	log.Println(fmt.Sprintf("Starting gateway server on :%d", g.port))
	g.server.Start()
	return nil
}

func (g *Gateway) Stop(ctx context.Context) error {
	log.Println("Stopping gateway server")
	return g.server.Stop(ctx)
}
