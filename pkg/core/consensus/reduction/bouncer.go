package reduction

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

type bouncer struct {
	publisher wire.EventPublisher
	handler   handler
}

func newBouncer(publisher wire.EventPublisher, handler handler) *bouncer {
	return &bouncer{
		publisher: publisher,
		handler:   handler,
	}
}

func (b *bouncer) repropagate(ev wire.Event) {
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
