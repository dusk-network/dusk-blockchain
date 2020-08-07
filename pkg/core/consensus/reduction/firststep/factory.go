package firststep

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/republisher"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

// Factory creates a first step reduction Component
type Factory struct {
	Bus         eventbus.Broker
	RBus        *rpcbus.RPCBus
	Keys        key.Keys
	timeout     time.Duration
	Republisher *republisher.Republisher
}

// NewFactory instantiates a Factory
func NewFactory(broker eventbus.Broker, rpcBus *rpcbus.RPCBus, keys key.Keys, timeout time.Duration) *Factory {
	reductionRepublisher := republisher.New(broker, topics.Reduction)
	if config.Get().General.SafeCallbackListener {
		reductionRepublisher = republisher.NewSafe(broker, topics.Reduction)
	}
	return &Factory{
		broker,
		rpcBus,
		keys,
		timeout,
		reductionRepublisher,
	}
}

// Instantiate a first step reduction Component
// Implements consensus.ComponentFactory.
func (f *Factory) Instantiate() consensus.Component {
	return NewComponent(f.Bus, f.RBus, f.Keys, f.timeout)
}

// CreateReducer is a reduction.FactoryFunc
func CreateReducer(broker *eventbus.EventBus, rpcBus *rpcbus.RPCBus, keys key.Keys, timeout time.Duration) reduction.Reducer {
	f := NewFactory(broker, rpcBus, keys, timeout)
	component := f.Instantiate()
	return component.(*Reducer)
}
