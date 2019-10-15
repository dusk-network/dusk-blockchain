package agreement

import (
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

type Factory struct {
	publisher    eventbus.Publisher
	keys         user.Keys
	workerAmount int
}

func NewFactory(publisher eventbus.Publisher, keys user.Keys) *Factory {
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
