package bus

import "github.com/fluxor-io/fluxor/pkg/reactor"

// ReactorProvider provides a way to get a reactor.
type ReactorProvider interface {
	// GetReactor returns a reactor.
	GetReactor() *reactor.Reactor
}
