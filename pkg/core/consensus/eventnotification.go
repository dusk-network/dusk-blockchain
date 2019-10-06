package consensus

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

func LaunchNotification(subscriber eventbus.Subscriber, deserializer wire.EventDeserializer, topic topics.Topic) <-chan wire.Event {
	notification := newNotification(deserializer)
	eventbus.NewTopicListener(subscriber, notification, topic, eventbus.ChannelType)
	return notification.reduChan
}

type notification struct {
	reduChan     chan wire.Event
	deserializer wire.EventDeserializer
}

func newNotification(deserializer wire.EventDeserializer) *notification {
	reduChan := make(chan wire.Event)
	return &notification{
		reduChan:     reduChan,
		deserializer: deserializer,
	}
}

func (n *notification) Collect(buf bytes.Buffer) error {
	// Copy the buffer, as multiple components will receive this same pointer
	ev, err := n.deserializer.Deserialize(&buf)
	if err != nil {
		return err
	}
	n.reduChan <- ev
	return nil
}
