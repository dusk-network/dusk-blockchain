package eventbus

import (
	"bytes"
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
	lg "github.com/sirupsen/logrus"
)

var _ Broker = (*EventBus)(nil)

var ringBufferLength = 2000
var napTime = 1 * time.Millisecond
var _signal struct{}
var logEB = lg.WithField("process", "eventbus")

// TopicProcessor is the interface for preprocessing events belonging to a specific topic
type (
	TopicProcessor interface {
		Process(*bytes.Buffer) (*bytes.Buffer, error)
	}
	// Preprocessor allows registration of preprocessors to be applied to incoming Event on a specific topic
	Preprocessor interface {
		RegisterPreprocessor(string, ...TopicProcessor) []uint32
		RemovePreprocessor(string, uint32)
		RemoveAllPreprocessors(string)
	}

	// Subscriber subscribes a channel to Event notifications on a specific topic
	Subscriber interface {
		Preprocessor
		Subscribe(string, chan<- bytes.Buffer) uint32
		SubscribeCallback(string, func(bytes.Buffer) error) uint32
		SubscribeStream(string, io.WriteCloser) uint32
		Unsubscribe(string, uint32)
		// RegisterPreprocessor(string, ...TopicProcessor)
	}

	// Publisher publishes serialized messages on a specific topic
	Publisher interface {
		Publish(string, bytes.Buffer)
		Stream(string, bytes.Buffer)
	}

	// Broker is an Publisher and an Subscriber
	Broker interface {
		Subscriber
		Publisher
	}

	// EventBus - box for handlers and callbacks.
	EventBus struct {
		busLock          sync.RWMutex
		handlers         *handlerMap
		callbackHandlers *handlerMap
		streamHandlers   *handlerMap
		preprocessors    map[string][]idTopicProcessor
	}

	idTopicProcessor struct {
		TopicProcessor
		id uint32
	}
)

// New returns new EventBus with empty handlers.
func New() *EventBus {
	return &EventBus{
		sync.RWMutex{},
		newHandlerMap(),
		newHandlerMap(),
		newHandlerMap(),
		make(map[string][]idTopicProcessor),
	}
}

// Subscribe subscribes to a topic with a channel.
func (bus *EventBus) Subscribe(topic string, messageChannel chan<- bytes.Buffer) uint32 {
	return bus.handlers.Store(topic, &channelHandler{messageChannel})
}

// SubscribeCallback subscribes to a topic with a callback.
func (bus *EventBus) SubscribeCallback(topic string, callback func(bytes.Buffer) error) uint32 {
	return bus.callbackHandlers.Store(topic, &callbackHandler{callback})
}

// SubscribeStream subscribes to a topic with a stream.
func (bus *EventBus) SubscribeStream(topic string, w io.WriteCloser) uint32 {
	bus.busLock.Lock()
	defer bus.busLock.Unlock()

	// Each streamHandler uses its own ringBuffer to collect topic events
	// Multiple-producers single-consumer approach utilizing a ringBuffer
	ringBuf := ring.NewBuffer(ringBufferLength)
	sh := &streamHandler{topic, ringBuf}

	// single-consumer
	_ = ring.NewConsumer(ringBuf, Consume, w)

	return bus.streamHandlers.Store(topic, sh)
}

// Consume an item by writing it to the specified WriteCloser
func Consume(items [][]byte, w io.WriteCloser) bool {
	for _, data := range items {
		if _, err := w.Write(data); err != nil {
			logEB.WithField("queue", "ringbuffer").WithError(err).Warnln("error in writing to WriteCloser")
			return false
		}
	}

	return true
}

// Unsubscribe removes a handler defined for a topic.
func (bus *EventBus) Unsubscribe(topic string, id uint32) {
	found := bus.handlers.Delete(topic, id) ||
		bus.callbackHandlers.Delete(topic, id) ||
		bus.streamHandlers.Delete(topic, id)

	logEB.WithFields(lg.Fields{
		"found": found,
		"topic": topic,
	}).Traceln("unsubscribing")
}

// RegisterPreprocessor will add a set of TopicProcessor to a specified topic.
func (bus *EventBus) RegisterPreprocessor(topic string, preprocessors ...TopicProcessor) []uint32 {
	pproc := make([]idTopicProcessor, len(preprocessors))
	pprocIds := make([]uint32, len(preprocessors))
	for i := 0; i < len(preprocessors); i++ {
		id := rand.Uint32()
		pprocIds[i] = id
		pproc[i] = idTopicProcessor{
			TopicProcessor: preprocessors[i],
			id:             id,
		}
	}

	bus.busLock.Lock()
	defer bus.busLock.Unlock()
	if _, ok := bus.preprocessors[topic]; ok {
		bus.preprocessors[topic] = append(bus.preprocessors[topic], pproc...)
		return pprocIds
	}

	bus.preprocessors[topic] = pproc
	return pprocIds
}

// RemovePreprocessor removes all TopicProcessor previously registered on a given topic using its ID
func (bus *EventBus) RemovePreprocessor(topic string, id uint32) {
	bus.busLock.Lock()
	defer bus.busLock.Unlock()
	if pprocs, ok := bus.preprocessors[topic]; ok {
		for idx, preproc := range pprocs {
			if preproc.id == id {
				// remove the item
				pprocs = append(pprocs[:idx], pprocs[idx+1:]...)
				bus.preprocessors[topic] = pprocs
				return
			}
		}
	}
}

// RemoveAllPreprocessors removes all TopicProcessor from a topic
func (bus *EventBus) RemoveAllPreprocessors(topic string) {
	bus.busLock.Lock()
	defer bus.busLock.Unlock()
	delete(bus.preprocessors, topic)
}

func (bus *EventBus) preprocess(topic string, messageBuffer *bytes.Buffer) (bytes.Buffer, error) {
	if preprocessors := bus.getPreprocessors(topic); len(preprocessors) > 0 {
		for _, preprocessor := range preprocessors {
			var err error
			messageBuffer, err = preprocessor.Process(messageBuffer)
			if err != nil {
				return bytes.Buffer{}, err
			}
		}
	}

	return *messageBuffer, nil
}

// Publish executes callback defined for a topic.
func (bus *EventBus) Publish(topic string, messageBuffer bytes.Buffer) {
	processedMsg, err := bus.preprocess(topic, &messageBuffer)
	if err != nil {
		logEB.WithError(err).Errorln("preprocessor error")
		return
	}

	if handlers := bus.handlers.Load(topic); handlers != nil {
		for _, handler := range handlers {
			if err := handler.Publish(processedMsg); err != nil {
				logEB.WithField("topic", topic).Warnln("handler.messageChannel buffer failed")
			}
		}
	}

	if callbackHandlers := bus.callbackHandlers.Load(topic); callbackHandlers != nil {
		for _, handler := range callbackHandlers {
			if err := handler.Publish(processedMsg); err != nil {
				logEB.WithError(err).WithField("topic", topic).Warnln("error when triggering callback")
			}
		}
	}
}

// Stream a buffer to the subscribers for a specific topic.
func (bus *EventBus) Stream(topic string, messageBuffer bytes.Buffer) {
	processedMsg, err := bus.preprocess(topic, &messageBuffer)
	if err != nil {
		logEB.WithError(err).WithField("topic", topic).Errorln("preprocessor error")
		return
	}

	// The handlers are simply a means to avoid memory leaks.
	handlers := bus.streamHandlers.Load(topic)
	for _, handler := range handlers {
		if err := handler.Publish(processedMsg); err != nil {
			logEB.WithError(err).WithField("topic", topic).Debugln("cannot publish event on stringhandler")
			continue
		}
	}
}

func (bus *EventBus) getPreprocessors(topic string) []idTopicProcessor {
	bus.busLock.RLock()
	processors := make([]idTopicProcessor, 0)
	if busProcessors, ok := bus.preprocessors[topic]; ok {
		processors = append(processors, busProcessors...)
	}

	bus.busLock.RUnlock()
	return processors
}
