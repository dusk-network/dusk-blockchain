package selection

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

type Factory struct {
	publisher eventbus.Publisher
	timeout   time.Duration
}

func NewFactory(publisher eventbus.Publisher, timeout time.Duration) *Factory {
	return &Factory{
		publisher,
		timeout,
	}
}

func (f *Factory) Instantiate() consensus.Component {
	return NewComponent(f.publisher, f.timeout)
}
