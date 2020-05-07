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
	ctx           context.Context
}

// NewFactory instantiates a Factory.
func NewFactory(ctx context.Context, broker eventbus.Broker, rpcBus *rpcbus.RPCBus, walletPrivKey *transactions.SecretKey, walletPubKey *transactions.PublicKey, proxy transactions.Proxy) *Factory {
	return &Factory{
		Bus:           broker,
		RBus:          rpcBus,
		walletPrivKey: walletPrivKey,
		walletPubKey:  walletPubKey,
		proxy:         proxy,
		ctx:           ctx,
	}
}

// Instantiate a candidate Generator.
// Implements consensus.ComponentFactory.
func (f *Factory) Instantiate() consensus.Component {
	return NewComponent(f.ctx, f.Bus, f.walletPrivKey, f.walletPubKey, f.RBus, f.proxy.BlockGenerator())
}
