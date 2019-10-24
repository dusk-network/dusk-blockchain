package reduction

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/key"
)

type Factory struct {
	publisher eventbus.Publisher
	rpcBus    *rpcbus.RPCBus
	keys      key.ConsensusKeys
	timeout   time.Duration
}

func NewFactory(publisher eventbus.Publisher, rpcBus *rpcbus.RPCBus, keys key.ConsensusKeys, timeout time.Duration) *Factory {
	return &Factory{
		publisher,
		rpcBus,
		keys,
		timeout,
	}
}

func (f *Factory) Instantiate() consensus.Component {
	return newComponent(f.publisher, f.rpcBus, f.keys, f.timeout)
}
