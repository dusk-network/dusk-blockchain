package candidate

import (
	"context"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

// Factory creates a candidate.Generator.
type Factory struct {
	Bus           eventbus.Broker
	RBus          *rpcbus.RPCBus
	walletPrivKey *transactions.SecretKey
	walletPubKey  *transactions.PublicKey
	proxy         transactions.Proxy
}

// NewFactory instantiates a Factory.
func NewFactory(broker eventbus.Broker, rpcBus *rpcbus.RPCBus, walletPrivKey *transactions.SecretKey, walletPubKey *transactions.PublicKey, proxy transactions.Proxy) *Factory {
	return &Factory{
		Bus:           broker,
		RBus:          rpcBus,
		walletPrivKey: walletPrivKey,
		walletPubKey:  walletPubKey,
		proxy:         proxy,
	}
}

// Instantiate a candidate Generator.
// Implements consensus.ComponentFactory.
func (f *Factory) Instantiate() consensus.Component {
	// TODO: Instantite needs a context
	return NewComponent(context.Background(), f.Bus, f.walletPrivKey, f.walletPubKey, f.RBus, f.proxy.BlockGenerator())
}
