package firststep

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/republisher"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/v2/key"
)

// Factory creates a first step reduction Component
type Factory struct {
	Bus         eventbus.Broker
	RBus        *rpcbus.RPCBus
	Keys        key.ConsensusKeys
	timeout     time.Duration
	Republisher *republisher.Republisher
}

// NewFactory instantiates a Factory
func NewFactory(broker eventbus.Broker, rpcBus *rpcbus.RPCBus, keys key.ConsensusKeys, timeout time.Duration) *Factory {
	r := republisher.New(broker, topics.Reduction)
	return &Factory{
		broker,
		rpcBus,
		keys,
		timeout,
		r,
	}
}

// Instantiate a first step reduction Component
// Implements consensus.ComponentFactory.
func (f *Factory) Instantiate() consensus.Component {
	return NewComponent(f.Bus, f.RBus, f.Keys, f.timeout)
}

// CreateReducer is a reduction.FactoryFunc
func CreateReducer(broker *eventbus.EventBus, rpcBus *rpcbus.RPCBus, keys key.ConsensusKeys, timeout time.Duration) reduction.Reducer {
	f := NewFactory(broker, rpcBus, keys, timeout)
	component := f.Instantiate()
	return component.(*Reducer)
}
