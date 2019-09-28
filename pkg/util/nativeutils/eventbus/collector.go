package eventbus

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	lg "github.com/sirupsen/logrus"
)

const (
	// Channel is the Listener type that relies on channels to communicate
	Channel ListenerType = iota
	// Callback is the Listener type that relies on Callbacks to notify
	// messages
	Callback
)

// QuitTopic is the topic to make all components quit
const QuitTopic = "quit"

// ListenerType is the enum of the type of available listeners
type ListenerType int

// TopicListener is a helper interface to connect an EventCollector to the
// EventBus.
// Considering that the EventCollector handles finite messages rather than
// packages, this interface is implemented only by callback and channel
// subscribers
type TopicListener interface {
	Quit()
}

// NewTopicListener creates a topic listener that subscribes an EventCollector
// to a Subscriber with a desidered Listener
func NewTopicListener(subscriber Subscriber, collector wire.EventCollector, topic string, listenerType ListenerType) TopicListener {
	switch listenerType {
	case Channel:
		return newChanCollector(subscriber, collector, topic)
	case Callback:
		return newCallbackCollector(subscriber, collector.Collect, topic)
	}
	return nil
}

type callbackCollector struct {
	Listener
	subscriber Subscriber
	id         uint32
	topic      string
}

func newCallbackCollector(subscriber Subscriber, callback func(bytes.Buffer) error, topic string) TopicListener {
	cbListener := NewCallbackListener(callback)
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
type chanCollector struct {
	subscriber     Subscriber
	eventCollector wire.EventCollector
	msgChan        <-chan bytes.Buffer
	msgChanID      uint32
	quitChan       chan bytes.Buffer
	quitChanID     uint32
	topic          string
	log            *lg.Entry
}

// NewTopicListener creates the TopicListener listening to a topic on the EventBus.
// The EventBus, EventCollector and Topic are injected
func newChanCollector(subscriber Subscriber, collector wire.EventCollector, topic string) TopicListener {
	msgChan := make(chan bytes.Buffer, 100)
	msgListener := NewChanListener(msgChan)

	quitChan := make(chan bytes.Buffer, 1)
	quitListener := NewChanListener(quitChan)

	tl := &chanCollector{
		subscriber:     subscriber,
		msgChan:        msgChan,
		msgChanID:      subscriber.Subscribe(topic, msgListener),
		quitChan:       quitChan,
		quitChanID:     subscriber.Subscribe(string(QuitTopic), quitListener),
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
			n.subscriber.Unsubscribe(string(QuitTopic), n.quitChanID)
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
