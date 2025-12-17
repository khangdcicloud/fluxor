package reactor

import (
	"context"

	"github.com/fluxor-io/fluxor/pkg/component"
)

type Reactor struct {
	component.Base
	name    string
	mailbox chan func()
}

func NewReactor(name string, size int) *Reactor {
	r := &Reactor{
		name:    name,
		mailbox: make(chan func(), size),
	}
	return r
}

func (r *Reactor) OnStart(ctx context.Context, bus any) {
	r.Go(r.loop)
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
