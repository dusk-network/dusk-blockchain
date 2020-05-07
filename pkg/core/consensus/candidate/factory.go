package candidate

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// Factory creates a candidate.Generator.
type Factory struct {
	Bus           eventbus.Broker
	RBus          *rpcbus.RPCBus
	walletPrivKey *transactions.SecretKey
	walletPubKey  *transactions.PublicKey
	rusk          rusk.RuskClient
}

// NewFactory instantiates a Factory.
func NewFactory(broker eventbus.Broker, rpcBus *rpcbus.RPCBus, walletPrivKey *transactions.SecretKey, walletPubKey *transactions.PublicKey, rusk rusk.RuskClient) *Factory {
	return &Factory{
		Bus:           broker,
		RBus:          rpcBus,
		walletPrivKey: walletPrivKey,
		walletPubKey:  walletPubKey,
		rusk:          rusk,
	}
}

// Instantiate a candidate Generator.
// Implements consensus.ComponentFactory.
func (f *Factory) Instantiate() consensus.Component {
	return NewComponent(f.Bus, f.walletPrivKey, f.walletPubKey, f.RBus, f.rusk)
}
