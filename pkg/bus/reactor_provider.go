package bus

import "github.com/example/goflux/pkg/reactor"

// ReactorProvider is an interface that provides a reactor for a given key.
// This is used to break the import cycle between the bus and the runtime.
type ReactorProvider interface {
	ReactorForKey(key string) *reactor.Reactor
}
