package secondstep

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/key"
)

// Factory creates a second step reduction Component
type Factory struct {
	Bus     eventbus.Broker
	RpcBus  *rpcbus.RPCBus
	Keys    key.ConsensusKeys
	timeout time.Duration
}

// NewFactory creates a Factory
func NewFactory(broker eventbus.Broker, rpcBus *rpcbus.RPCBus, keys key.ConsensusKeys, timeout time.Duration) *Factory {
	return &Factory{
		broker,
		rpcBus,
		keys,
		timeout,
	}
}

// Instantiate a second step reduction Component
func (f *Factory) Instantiate() consensus.Component {
	return NewComponent(f.Bus, f.RpcBus, f.Keys, f.timeout)
}
