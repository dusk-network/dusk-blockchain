package wire

import (
	"bytes"
	"io"
	"math/rand"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/ring"
)

var _ EventBroker = (*EventBus)(nil)

var ringBufferLength = 200
var napTime = 1 * time.Millisecond
var _signal struct{}

// EventBus - box for handlers and callbacks.
type EventBus struct {
	busLock          sync.RWMutex
	handlers         *handlerMap
	callbackHandlers *handlerMap
	streamHandlers   *handlerMap
	preprocessors    map[string][]idTopicProcessor
	ringbuffer       *ring.Buffer
}

type idTopicProcessor struct {
	TopicProcessor
	id uint32
}

// NewEventBus returns new EventBus with empty handlers.
func NewEventBus() *EventBus {
	return &EventBus{
		sync.RWMutex{},
		newHandlerMap(),
		newHandlerMap(),
		newHandlerMap(),
		make(map[string][]idTopicProcessor),
		ring.NewBuffer(ringBufferLength),
	}
}

// subscribe handles the subscription logic and is utilized by the public
// Subscribe functions
func (bus *EventBus) subscribe(topic string, handler *channelHandler) {
	bus.handlers.Store(topic, handler)
}

// Subscribe subscribes to a topic with a channel.
func (bus *EventBus) Subscribe(topic string, messageChannel chan<- *bytes.Buffer) uint32 {
	id := rand.Uint32()
	bus.subscribe(topic, &channelHandler{
		id, messageChannel,
	})

	return id
}

func (bus *EventBus) subscribeCallback(topic string, handler *callbackHandler) {
	bus.callbackHandlers.Store(topic, handler)
}

// SubscribeCallback subscribes to a topic with a callback.
func (bus *EventBus) SubscribeCallback(topic string, callback func(*bytes.Buffer) error) uint32 {
	id := rand.Uint32()
	bus.subscribeCallback(topic, &callbackHandler{
		id, callback,
	})

	return id
}

func (bus *EventBus) subscribeStream(topic string, handler *streamHandler) {
	bus.streamHandlers.Store(topic, handler)
}

func newStreamHandler(id uint32, topic string) *streamHandler {
	exitChan := make(chan struct{})
	return &streamHandler{id, exitChan, topic}
}

// Pipe to a WriteCloser. If all messages are consumed, the process stops to give the producer a chance to produce more messages
func (s *streamHandler) Pipe(c *ring.Consumer, w io.WriteCloser) {
	for {
		select {
		case <-s.exitChan:
			if err := w.Close(); err != nil {
				log.WithFields(log.Fields{
					"process": "eventbus",
					"topic":   s.topic,
				}).WithError(err).Warnln("error in closing the WriteCloser")
			}
			return
		default:
			data, duplicate := c.Consume()
			if !duplicate {
				if _, err := w.Write(data); err != nil {
					log.WithFields(log.Fields{
						"process": "eventbus",
						"topic":   s.topic,
					}).WithError(err).Warnln("error in writing to WriteCloser")
					s.exitChan <- _signal
				}
				continue
			}
			time.Sleep(napTime)
		}
	}
}

// SubscribeStream subscribes to a topic with a stream.
func (bus *EventBus) SubscribeStream(topic string, w io.WriteCloser) uint32 {
	id := rand.Uint32()
	c := ring.NewConsumer(bus.ringbuffer)
	sh := newStreamHandler(id, topic)
	bus.busLock.Lock()
	defer bus.busLock.Unlock()
	bus.subscribeStream(topic, sh)

	go sh.Pipe(c, w)
	return id
}

// Unsubscribe removes a handler defined for a topic.
// Returns true if a handler is found with the id and the topic specified
func (bus *EventBus) Unsubscribe(topic string, id uint32) {
	bus.handlers.Delete(topic, id)
	bus.callbackHandlers.Delete(topic, id)
	bus.streamHandlers.Delete(topic, id)
}

// RegisterPreprocessor will add a set of preprocessors to a specified topic.
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

func (bus *EventBus) RemoveAllPreprocessors(topic string) {
	bus.busLock.Lock()
	defer bus.busLock.Unlock()
	delete(bus.preprocessors, topic)
}

func (bus *EventBus) preprocess(topic string, messageBuffer *bytes.Buffer) (*bytes.Buffer, error) {
	if preprocessors := bus.getPreprocessors(topic); len(preprocessors) > 0 {
		for _, preprocessor := range preprocessors {
			var err error
			messageBuffer, err = preprocessor.Process(messageBuffer)
			if err != nil {
				return nil, err
			}
		}
	}

	return messageBuffer, nil
}

// Publish executes callback defined for a topic.
func (bus *EventBus) Publish(topic string, messageBuffer *bytes.Buffer) {
	processedMsg, err := bus.preprocess(topic, messageBuffer)
	if err != nil {
		log.WithFields(log.Fields{
			"process": "eventbus",
			"error":   err,
		}).Errorln("preprocessor error")
		return
	}

	if handlers := bus.handlers.Load(topic); handlers != nil {
		for _, handler := range handlers {
			if err := handler.Publish(processedMsg); err != nil {
				log.WithFields(log.Fields{
					"topic":   topic,
					"process": "eventbus",
				}).Warnln("handler.messageChannel buffer failed")
			}
		}
	}

	if callbackHandlers := bus.callbackHandlers.Load(topic); callbackHandlers != nil {
		for _, handler := range callbackHandlers {
			if err := handler.Publish(processedMsg); err != nil {
				log.WithFields(log.Fields{
					"topic":   topic,
					"process": "event bus",
					"error":   err,
				}).Errorln("error when triggering callback")
			}
		}
	}
}

// Stream a buffer to the subscribers for a specific topic.
func (bus *EventBus) Stream(topic string, messageBuffer *bytes.Buffer) {
	processedMsg, err := bus.preprocess(topic, messageBuffer)
	if err != nil {
		log.WithFields(log.Fields{
			"process": "eventbus",
			"error":   err,
		}).Errorln("preprocessor error")
		return
	}

	if handlers := bus.streamHandlers.Load(topic); handlers != nil {
		bus.ringbuffer.Put(processedMsg.Bytes())
	}
}

func copyBuffer(m *bytes.Buffer) *bytes.Buffer {
	var mCopy bytes.Buffer
	if m != nil {
		mCopy = *m
	}

	return &mCopy
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
