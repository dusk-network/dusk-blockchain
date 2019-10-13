package agreement

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

type Factory struct {
	publisher eventbus.Publisher
	keys      user.Keys
}

func NewFactory(publisher eventbus.Publisher, keys user.Keys) *Factory {
	return &Factory{
		publisher,
		keys,
	}
}

func (f *Factory) Instantiate() Component {
	return newComponent(f.publisher, f.keys)
}
