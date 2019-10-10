package reduction

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

type Factory struct {
	publisher         eventbus.Publisher
	rpcBus            *rpcbus.RPCBus
	keys              user.Keys
	requestStepUpdate func()
}

func NewFactory(publisher eventbus.Publisher, rpcBus rpcbus.RPCBus, keys user.Keys, requestStepUpdate func()) *Factory {
	return &Factory{
		publisher,
		rpcBus,
		keys,
		requestStepUpdate,
	}
}

func (f *Factory) Instantiate() Component {
	return newComponent(f.publisher, f.rpcBus, keys)
}
