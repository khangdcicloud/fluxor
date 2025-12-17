package gateway

import (
	"context"
	"github.com/example/goflux/pkg/bus"
	"github.com/example/goflux/pkg/httpx"
	"github.com/example/goflux/pkg/reactor"
	"log"
)

type Gateway struct {
	server *httpx.Server
}

func NewGateway(reactor *reactor.Reactor, bus bus.Bus) *Gateway {
	g := &Gateway{}

	// Use httpx.NewServer from our shared package
	g.server = httpx.NewServer(8080, reactor, bus)

	return g
}

func (g *Gateway) Start(ctx context.Context) error {
	log.Println("Starting gateway server on :8080")
	g.server.Start()
	return nil
}

func (g *Gateway) Stop(ctx context.Context) error {
	log.Println("Stopping gateway server")
	return g.server.Stop(ctx)
}
