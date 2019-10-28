package firststep

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/key"
)

// Factory creates a first step reduction Component
type Factory struct {
	Bus     eventbus.Broker
	RBus    *rpcbus.RPCBus
	Keys    key.ConsensusKeys
	timeout time.Duration
}

// NewFactory instantiates a Factory
func NewFactory(broker eventbus.Broker, rpcBus *rpcbus.RPCBus, keys key.ConsensusKeys, timeout time.Duration) *Factory {
	return &Factory{
		broker,
		rpcBus,
		keys,
		timeout,
	}
}

// Instantiate a first step reduction Component
func (f *Factory) Instantiate() reduction.Reducer {
	return NewComponent(f.Bus, f.RBus, f.Keys, f.timeout)
}

// CreateReducer is a reduction.FactoryFunc
func CreateReducer(broker *eventbus.EventBus, rpcBus *rpcbus.RPCBus, keys key.ConsensusKeys, timeout time.Duration) reduction.Reducer {
	f := NewFactory(broker, rpcBus, keys, timeout)
	component := f.Instantiate()
	return component.(*Reducer)
}
