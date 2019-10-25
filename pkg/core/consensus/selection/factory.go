package selection

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

type Factory struct {
	Bus     eventbus.Broker
	timeout time.Duration
}

func NewFactory(bus eventbus.Broker, timeout time.Duration) *Factory {
	return &Factory{
		bus,
		timeout,
	}
}

func (f *Factory) Instantiate() consensus.Component {
	return NewComponent(f.Bus, f.timeout)
}
