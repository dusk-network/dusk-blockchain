package agreement

import (
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/key"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/republisher"
)

// Factory creates the agreement component.
type Factory struct {
	broker       eventbus.Broker
	keys         key.ConsensusKeys
	workerAmount int
	Republisher  *republisher.Republisher
}

// NewFactory instantiates a Factory.
func NewFactory(broker eventbus.Broker, keys key.ConsensusKeys) *Factory {
	amount := cfg.Get().Performance.AccumulatorWorkers
	r := republisher.New(broker, topics.Agreement)

	return &Factory{
		broker:       broker,
		keys:         keys,
		workerAmount: amount,
		Republisher:  r,
	}
}

// Instantiate an agreement component and return it.
// Implements consensus.ComponentFactory.
func (f *Factory) Instantiate() consensus.Component {
	return newComponent(f.broker, f.keys, f.workerAmount)
}
