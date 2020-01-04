package republisher

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

// Republisher handles the repropagation of Agreement messages
type Republisher struct {
	tpc    topics.Topic
	broker eventbus.Broker
	id     uint32
}

// New creates a Republisher
func New(eb eventbus.Broker, tpc topics.Topic) *Republisher {
	r := &Republisher{
		broker: eb,
		tpc:    tpc,
	}
	r.id = r.Activate()
	return r
}

// Stop a Republisher
func (r *Republisher) Stop() {
	r.broker.Unsubscribe(r.tpc, r.id)
}

// Activate the Republisher by listening to Agreement messages through a
// callback listener
func (r *Republisher) Activate() uint32 {
	if r.id != 0 {
		return r.id
	}

	l := eventbus.NewCallbackListener(r.Republish)
	r.id = r.broker.Subscribe(r.tpc, l)
	return r.id
}

// Republish intercepts an Agreement message and repropagates it immediately
func (r *Republisher) Republish(b bytes.Buffer) error {
	if err := topics.Prepend(&b, r.tpc); err != nil {
		return err
	}

	r.broker.Publish(topics.Gossip, &b)
	return nil
}
