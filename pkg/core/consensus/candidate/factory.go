package candidate

import (
	"context"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

// Factory creates a candidate.Generator.
type Factory struct {
	Bus          eventbus.Broker
	RBus         *rpcbus.RPCBus
	walletPubKey *keys.PublicKey
	ctx          context.Context
}

// NewFactory instantiates a Factory.
func NewFactory(ctx context.Context, broker eventbus.Broker, rpcBus *rpcbus.RPCBus, walletPubKey *keys.PublicKey) *Factory {
	return &Factory{
		Bus:          broker,
		RBus:         rpcBus,
		walletPubKey: walletPubKey,
		ctx:          ctx,
	}
}

// Instantiate a candidate Generator.
// Implements consensus.ComponentFactory.
func (f *Factory) Instantiate() consensus.Component {
	return NewComponent(f.ctx, f.Bus, f.walletPubKey, f.RBus)
}
