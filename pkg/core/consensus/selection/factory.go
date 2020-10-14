package selection

import (
	"context"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

// Factory creates the selection component.
type Factory struct {
	Bus     eventbus.Broker
	timeout time.Duration
	proxy   transactions.Proxy
	ctx     context.Context
}

// NewFactory instantiates a Factory.
func NewFactory(ctx context.Context, bus eventbus.Broker, timeout time.Duration, proxy transactions.Proxy) *Factory {
	return &Factory{
		bus,
		timeout,
		proxy,
		ctx,
	}
}

// Instantiate a Selector and return it.
// Implements consensus.ComponentFactory.
func (f *Factory) Instantiate() consensus.Component {
	return NewComponent(f.ctx, f.Bus, f.timeout, f.proxy.Provisioner())
}
