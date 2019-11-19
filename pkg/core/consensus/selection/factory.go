package selection

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

// Factory creates the selection component.
type Factory struct {
	Bus     eventbus.Broker
	timeout time.Duration
}

// NewFactory instantiates a Factory.
func NewFactory(bus eventbus.Broker, timeout time.Duration) *Factory {
	return &Factory{
		bus,
		timeout,
	}
}

// Instantiate a Selector and return it.
// Implements consensus.ComponentFactory.
func (f *Factory) Instantiate() consensus.Component {
	return NewComponent(f.Bus, f.timeout)
}
