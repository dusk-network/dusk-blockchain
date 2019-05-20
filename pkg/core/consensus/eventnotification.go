package consensus

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func LaunchNotification(eventbus wire.EventSubscriber, deserializer wire.EventDeserializer, topic string) <-chan wire.Event {
	notification := newNotification(deserializer)
	listener := wire.NewTopicListener(eventbus, notification, topic)
	go listener.Accept()
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

func (n *notification) Collect(buf *bytes.Buffer) error {
	// Copy the buffer, as multiple components will receive this same pointer
	copyBuf := *buf
	ev, err := n.deserializer.Deserialize(&copyBuf)
	if err != nil {
		return err
	}
	n.reduChan <- ev
	return nil
}
