package agreement

import (
	"bytes"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-wallet/key"
)

// Factory creates the agreement component.
type Factory struct {
	broker       eventbus.Broker
	keys         key.ConsensusKeys
	workerAmount int
	r            *Republisher
}

// Republisher handles the repropagation of Agreement messages
type Republisher struct {
	broker eventbus.Broker
	id     uint32
}

// NewFactory instantiates a Factory.
func NewFactory(broker eventbus.Broker, keys key.ConsensusKeys) *Factory {
	amount := cfg.Get().Performance.AccumulatorWorkers
	r := newRepublisher(broker)

	return &Factory{
		broker,
		keys,
		amount,
		r,
	}
}

// Instantiate an agreement component and return it.
// Implements consensus.ComponentFactory.
func (f *Factory) Instantiate() consensus.Component {
	return newComponent(f.broker, f.keys, f.workerAmount)
}

// newRepublisher creates a Republisher
func newRepublisher(eb eventbus.Broker) *Republisher {
	r := &Republisher{broker: eb}
	r.id = r.Activate()
	return r
}

// Stop a Republisher
func (r *Republisher) Stop() {
	r.broker.Unsubscribe(topics.Agreement, r.id)
}

// Activate the Republisher by listening to Agreement messages through a
// callback listener
func (r *Republisher) Activate() uint32 {
	l := eventbus.NewCallbackListener(r.Republish)
	return r.broker.Subscribe(topics.Agreement, l)
}

// Republish intercepts an Agreement message and repropagates it immediately
func (r *Republisher) Republish(b bytes.Buffer) error {
	if err := topics.Prepend(&b, topics.Agreement); err != nil {
		return err
	}

	r.broker.Publish(topics.Gossip, &b)
	return nil
}
