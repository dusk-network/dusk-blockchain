package firststep

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/key"
)

type Factory struct {
	broker  eventbus.Broker
	rpcBus  *rpcbus.RPCBus
	keys    key.ConsensusKeys
	timeout time.Duration
}

func NewFactory(broker eventbus.Broker, rpcBus *rpcbus.RPCBus, keys key.ConsensusKeys, timeout time.Duration) *Factory {
	return &Factory{
		broker,
		rpcBus,
		keys,
		timeout,
	}
}

func (f *Factory) Instantiate() consensus.Component {
	return NewComponent(f.broker, f.rpcBus, f.keys, f.timeout)
}
