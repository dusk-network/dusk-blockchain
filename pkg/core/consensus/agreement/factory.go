package agreement

import (
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-wallet/key"
)

// Factory creates the agreement component.
type Factory struct {
	broker       eventbus.Broker
	keys         key.ConsensusKeys
	workerAmount int
}

// NewFactory instantiates a Factory.
func NewFactory(broker eventbus.Broker, keys key.ConsensusKeys) *Factory {
	amount := cfg.Get().Performance.AccumulatorWorkers
	return &Factory{
		broker,
		keys,
		amount,
	}
}

// Instantiate an agreement component and return it.
// Implements consensus.ComponentFactory.
func (f *Factory) Instantiate() consensus.Component {
	return newComponent(f.broker, f.keys, f.workerAmount)
}
