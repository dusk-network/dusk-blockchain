package eventbus

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	lg "github.com/sirupsen/logrus"
)

const (
	// ChannelType is the Listener type that relies on channels to communicate
	ChannelType ListenerType = iota
	// CallbackType is the Listener type that relies on Callbacks to notify
	// messages
	CallbackType
)

// ListenerType is the enum of the type of available listeners
type ListenerType int

// TopicListener is a helper interface to connect an EventCollector to the
// EventBus.
// Considering that the EventCollector handles finite messages rather than
// packages, this interface is implemented only by callback and channel
// subscribers
// Deprecated: use eventbus.NewChanListener or eventbus.NewCallbackListener instead
type TopicListener interface {
	Quit()
}

// NewTopicListener creates a topic listener that subscribes an EventCollector
// to a Subscriber with a desidered Listener
func NewTopicListener(subscriber Subscriber, collector wire.EventCollector, topic topics.Topic, listenerType ListenerType) TopicListener {
	switch listenerType {
	case ChannelType:
		return NewChanTopicListener(subscriber, collector, topic)
	case CallbackType:
		return NewCallbackTopicListener(subscriber, collector, topic)
	}
	return nil
}

type callbackCollector struct {
	Listener
	subscriber Subscriber
	id         uint32
	topic      topics.Topic
}

func NewCallbackTopicListener(subscriber Subscriber, collector wire.EventCollector, topic topics.Topic) TopicListener {
	cbListener := NewCallbackListener(collector.Collect)
	id := subscriber.Subscribe(topic, cbListener)
	return &callbackCollector{
		Listener: cbListener,
		id:       id,
		topic:    topic,
	}
}

// Quit unsubscribes the collector
func (c *callbackCollector) Quit() {
	c.subscriber.Unsubscribe(c.topic, c.id)
}

// channelCollector accepts events from the EventBus and takes care of reacting on
// quit Events. It delegates the business logic to the EventCollector which is
// supposed to handle the incoming events
// Deprecated: when the `Collect` callback includes a channel, we incur in potential backpressure because of the potentially different bufferization strategy of the `chanCollector.msgChan`
type chanCollector struct {
	subscriber     Subscriber
	eventCollector wire.EventCollector
	msgChan        <-chan bytes.Buffer
	msgChanID      uint32
	quitChan       chan bytes.Buffer
	quitChanID     uint32
	topic          topics.Topic
	log            *lg.Entry
}

// NewCallbackTopicListener creates the TopicListener listening to a topic on the EventBus.
// The EventBus, EventCollector and Topic are injected
func NewChanTopicListener(subscriber Subscriber, collector wire.EventCollector, topic topics.Topic) TopicListener {
	msgChan := make(chan bytes.Buffer, 100)
	msgListener := NewChanListener(msgChan)

	quitChan := make(chan bytes.Buffer, 1)
	quitListener := NewChanListener(quitChan)

	tl := &chanCollector{
		subscriber:     subscriber,
		msgChan:        msgChan,
		msgChanID:      subscriber.Subscribe(topic, msgListener),
		quitChan:       quitChan,
		quitChanID:     subscriber.Subscribe(topics.Quit, quitListener),
		eventCollector: collector,
		log: lg.WithFields(lg.Fields{
			"topic":   topic,
			"process": "channelCollector",
		}),
	}

	go tl.accept()
	return tl
}

// Accept incoming (mashalled) Events on the topic of interest and dispatch them to the EventCollector.Collect.
func (n *chanCollector) accept() {
	for {
		select {
		case <-n.quitChan:
			n.subscriber.Unsubscribe(n.topic, n.msgChanID)
			n.subscriber.Unsubscribe(topics.Quit, n.quitChanID)
			return

		case eventBuffer := <-n.msgChan:
			if len(n.msgChan) > 10 {
				n.log.WithField("Unconsumed", len(n.msgChan)).Debugln("Channel is accumulating messages")
			}
			if err := n.eventCollector.Collect(eventBuffer); err != nil {
				n.log.WithError(err).Errorln("Error in eventCollector.Collect")
			}
		}
	}
}

// Quit will kill the goroutine spawned by Accept, and unsubscribe from it's subscribed topics.
func (n *chanCollector) Quit() {
	n.quitChan <- bytes.Buffer{}
}
