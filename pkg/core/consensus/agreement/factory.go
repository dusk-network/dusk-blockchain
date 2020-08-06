package agreement

import (
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/republisher"
)

// Factory creates the agreement component.
type Factory struct {
	broker       eventbus.Broker
	keys         key.Keys
	workerAmount int
	Republisher  *republisher.Republisher
}

// NewFactory instantiates a Factory.
func NewFactory(broker eventbus.Broker, keys key.Keys) *Factory {
	amount := cfg.Get().Performance.AccumulatorWorkers
	agreementRepublisher := republisher.New(broker, topics.Agreement)
	if config.Get().General.SafeCallbackListener {
		agreementRepublisher = republisher.NewSafe(broker, topics.Agreement)
	}
	return &Factory{
		broker:       broker,
		keys:         keys,
		workerAmount: amount,
		Republisher:  agreementRepublisher,
	}
}

// Instantiate an agreement component and return it.
// Implements consensus.ComponentFactory.
func (f *Factory) Instantiate() consensus.Component {
	return newComponent(f.broker, f.keys, f.workerAmount)
}
