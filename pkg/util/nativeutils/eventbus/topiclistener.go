package eventbus

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	lg "github.com/sirupsen/logrus"
)

// QuitTopic is the topic to make all components quit
const QuitTopic = "quit"

// TopicListener accepts events from the EventBus and takes care of reacting on
// quit Events. It delegates the business logic to the EventCollector which is
// supposed to handle the incoming events
type TopicListener struct {
	subscriber     Subscriber
	eventCollector wire.EventCollector
	msgChan        <-chan bytes.Buffer
	msgChanID      uint32
	quitChan       chan bytes.Buffer
	quitChanID     uint32
	topic          string
}

// NewTopicListener creates the TopicListener listening to a topic on the EventBus.
// The EventBus, EventCollector and Topic are injected
func NewTopicListener(subscriber Subscriber, collector wire.EventCollector, topic string,
	preprocessors ...TopicProcessor) *TopicListener {

	msgChan := make(chan bytes.Buffer, 100)
	quitChan := make(chan bytes.Buffer, 1)
	msgChanID := subscriber.Subscribe(topic, msgChan)
	quitChanID := subscriber.Subscribe(string(QuitTopic), quitChan)

	if len(preprocessors) > 0 {
		subscriber.RegisterPreprocessor(topic, preprocessors...)
	}

	return &TopicListener{
		subscriber:     subscriber,
		msgChan:        msgChan,
		msgChanID:      msgChanID,
		quitChan:       quitChan,
		quitChanID:     quitChanID,
		topic:          topic,
		eventCollector: collector,
	}
}

func (n *TopicListener) log() *lg.Entry {
	return lg.WithFields(lg.Fields{
		"topic":   n.topic,
		"process": "TopicListener",
	})
}

// Accept incoming (mashalled) Events on the topic of interest and dispatch them to the EventCollector.Collect.
func (n *TopicListener) Accept() {
	log := n.log().WithField("id", n.msgChanID)
	log.Debugln("Accepting messages")
	for {
		select {
		case <-n.quitChan:
			n.subscriber.Unsubscribe(n.topic, n.msgChanID)
			n.subscriber.Unsubscribe(string(QuitTopic), n.quitChanID)
			return
		case eventBuffer := <-n.msgChan:
			if len(n.msgChan) > 10 {
				log.WithField("Unconsumed", len(n.msgChan)).Debugln("Channel is accumulating messages")
			}
			if err := n.eventCollector.Collect(eventBuffer); err != nil {
				log.WithError(err).Errorln("Error in eventCollector.Collect")
			}
		}
	}
}

// Quit will kill the goroutine spawned by Accept, and unsubscribe from it's subscribed topics.
func (n *TopicListener) Quit() {
	n.quitChan <- bytes.Buffer{}
}
