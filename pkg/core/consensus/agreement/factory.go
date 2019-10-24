package agreement

import (
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-wallet/key"
)

type Factory struct {
	publisher    eventbus.Publisher
	keys         key.ConsensusKeys
	workerAmount int
}

func NewFactory(publisher eventbus.Publisher, keys key.ConsensusKeys) *Factory {
	amount := cfg.Get().Performance.AccumulatorWorkers
	return &Factory{
		publisher,
		keys,
		amount,
	}
}

func (f *Factory) Instantiate() consensus.Component {
	return newComponent(f.publisher, f.keys, f.workerAmount)
}
