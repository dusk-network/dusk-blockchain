package candidate

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/key"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

// Factory creates a candidate.Generator.
type Factory struct {
	Bus          eventbus.Broker
	RBus         *rpcbus.RPCBus
	walletPubKey *key.PublicKey
}

// NewFactory instantiates a Factory.
func NewFactory(broker eventbus.Broker, rpcBus *rpcbus.RPCBus, walletPubKey *key.PublicKey) *Factory {
	return &Factory{
		Bus:          broker,
		RBus:         rpcBus,
		walletPubKey: walletPubKey,
	}
}

// Instantiate a candidate Generator.
// Implements consensus.ComponentFactory.
func (f *Factory) Instantiate() consensus.Component {
	return NewComponent(f.Bus, f.walletPubKey, f.RBus)
}
