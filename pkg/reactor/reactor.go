package reactor

import (
	"context"

	"github.com/fluxor-io/fluxor/pkg/types"
)

type Reactor struct {
	name    string
	mailbox chan func()
	bus     types.Bus
}

func NewReactor(name string, size int) *Reactor {
	r := &Reactor{
		name:    name,
		mailbox: make(chan func(), size),
	}
	return r
}

func (r *Reactor) Name() string {
	return r.name
}

func (r *Reactor) OnStart(ctx context.Context, bus types.Bus) error {
	r.bus = bus
	go r.loop(ctx)
	return nil
}

func (r *Reactor) OnStop(ctx context.Context) error {
	return nil
}

func (r *Reactor) Execute(fn func()) {
	r.mailbox <- fn
}

func (r *Reactor) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case fn := <-r.mailbox:
			fn()
		}
	}
}
