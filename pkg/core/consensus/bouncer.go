package consensus

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

type Bouncer struct {
	publisher wire.EventPublisher
	handler   EventHandler
}

func NewBouncer(publisher wire.EventPublisher, handler EventHandler) *Bouncer {
	return &Bouncer{
		publisher: publisher,
		handler:   handler,
	}
}

func (b *Bouncer) repropagate(ev wire.Event) {
	buf := new(bytes.Buffer)
	if err := b.handler.MarshalEdFields(buf, ev); err != nil {
		panic(err)
	}
	if err := b.handler.Marshal(buf, ev); err != nil {
		panic(err)
	}
	message, err := wire.AddTopic(buf, topics.BlockReduction)
	if err != nil {
		panic(err)
	}
	b.publisher.Publish(string(topics.Gossip), message)
}
