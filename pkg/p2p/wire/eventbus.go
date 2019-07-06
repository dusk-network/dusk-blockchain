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
var napTime = 10 * time.Millisecond
var _signal struct{}

// EventBus - box for handlers and callbacks.
type EventBus struct {
	busLock          sync.RWMutex
	handlers         map[string][]*channelHandler
	callbackHandlers map[string][]*callbackHandler
	streamHandlers   map[string][]*streamHandler
	preprocessors    map[string][]idTopicProcessor
	ringbuffer       *ring.Buffer
}

type callbackHandler struct {
	id       uint32
	callback func(*bytes.Buffer) error
}

type streamHandler struct {
	id       uint32
	exitChan chan struct{}
	topic    string
}

type channelHandler struct {
	id             uint32
	messageChannel chan<- *bytes.Buffer
}

type idTopicProcessor struct {
	TopicProcessor
	id uint32
}

// NewEventBus returns new EventBus with empty handlers.
func NewEventBus() *EventBus {
	return &EventBus{
		sync.RWMutex{},
		make(map[string][]*channelHandler),
		make(map[string][]*callbackHandler),
		make(map[string][]*streamHandler),
		make(map[string][]idTopicProcessor),
		ring.NewBuffer(ringBufferLength),
	}
}

// subscribe handles the subscription logic and is utilized by the public
// Subscribe functions
func (bus *EventBus) subscribe(topic string, handler *channelHandler) {
	bus.handlers[topic] = append(bus.handlers[topic], handler)
}

// Subscribe subscribes to a topic with a channel.
func (bus *EventBus) Subscribe(topic string, messageChannel chan<- *bytes.Buffer) uint32 {
	id := rand.Uint32()
	bus.busLock.Lock()
	defer bus.busLock.Unlock()
	bus.subscribe(topic, &channelHandler{
		id, messageChannel,
	})

	return id
}

func (bus *EventBus) subscribeCallback(topic string, handler *callbackHandler) {
	bus.callbackHandlers[topic] = append(bus.callbackHandlers[topic], handler)
}

// SubscribeCallback subscribes to a topic with a callback.
func (bus *EventBus) SubscribeCallback(topic string, callback func(*bytes.Buffer) error) uint32 {
	id := rand.Uint32()
	bus.busLock.Lock()
	defer bus.busLock.Unlock()
	bus.subscribeCallback(topic, &callbackHandler{
		id, callback,
	})

	return id
}

func (bus *EventBus) subscribeStream(topic string, handler *streamHandler) {
	bus.streamHandlers[topic] = append(bus.streamHandlers[topic], handler)
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
			// giving enough time to the producer to send stuff
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
func (bus *EventBus) Unsubscribe(topic string, id uint32) bool {
	bus.busLock.Lock()
	defer bus.busLock.Unlock()
	return bus.unsubscribe(topic, id)
}

func (bus *EventBus) unsubscribe(topic string, id uint32) bool {
	if _, ok := bus.handlers[topic]; ok {
		for i, handler := range bus.handlers[topic] {
			if handler.id == id {
				bus.handlers[topic] = append(bus.handlers[topic][:i],
					bus.handlers[topic][i+1:]...)
				return true
			}
		}
	}

	if _, ok := bus.callbackHandlers[topic]; ok {
		for i, handler := range bus.callbackHandlers[topic] {
			if handler.id == id {
				bus.callbackHandlers[topic] = append(bus.callbackHandlers[topic][:i],
					bus.callbackHandlers[topic][i+1:]...)
				return true
			}
		}
	}

	if _, ok := bus.streamHandlers[topic]; ok {
		for i, handler := range bus.streamHandlers[topic] {
			if handler.id == id {
				bus.streamHandlers[topic] = append(bus.streamHandlers[topic][:i],
					bus.streamHandlers[topic][i+1:]...)
				return true
			}
		}
	}

	return false
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

	if handlers := bus.getChannelHandlers(topic); len(handlers) > 0 {
		bus.publish(handlers, processedMsg, topic)
	}

	if callbackHandlers := bus.getCallbackHandlers(topic); len(callbackHandlers) > 0 {
		bus.publishCallback(callbackHandlers, processedMsg, topic)
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

	if handlers := bus.getStreamHandlers(topic); len(handlers) > 0 {
		bus.ringbuffer.Put(processedMsg.Bytes())
	}
}

func (bus *EventBus) publish(handlers []*channelHandler, messageBuffer *bytes.Buffer, topic string) {
	for _, handler := range handlers {
		mCopy := copyBuffer(messageBuffer)
		select {
		case handler.messageChannel <- mCopy:
		default:
			log.WithFields(log.Fields{
				"id":       handler.id,
				"topic":    topic,
				"process":  "eventbus",
				"capacity": len(handler.messageChannel),
			}).Warnln("handler.messageChannel buffer failed")
		}
	}
}

func (bus *EventBus) publishCallback(handlers []*callbackHandler, message *bytes.Buffer, topic string) {
	for _, handler := range handlers {
		mCopy := copyBuffer(message)
		if err := handler.callback(mCopy); err != nil {
			log.WithFields(log.Fields{
				"id":      handler.id,
				"topic":   topic,
				"process": "event bus",
				"error":   err,
			}).Errorln("error when triggering callback")
		}
	}
}

func copyBuffer(m *bytes.Buffer) *bytes.Buffer {
	var mCopy bytes.Buffer
	if m != nil {
		mCopy = *m
	}

	return &mCopy
}

func (bus *EventBus) getChannelHandlers(topic string) []*channelHandler {
	bus.busLock.RLock()
	defer bus.busLock.RUnlock()
	handlers := make([]*channelHandler, 0)
	if busHandlers, ok := bus.handlers[topic]; ok {
		handlers = append(handlers, busHandlers...)
	}

	return handlers
}

func (bus *EventBus) getCallbackHandlers(topic string) []*callbackHandler {
	bus.busLock.RLock()
	defer bus.busLock.RUnlock()
	handlers := make([]*callbackHandler, 0)
	if busHandlers, ok := bus.callbackHandlers[topic]; ok {
		handlers = append(handlers, busHandlers...)
	}

	return handlers
}

func (bus *EventBus) getStreamHandlers(topic string) []*streamHandler {
	bus.busLock.RLock()
	defer bus.busLock.RUnlock()
	handlers := make([]*streamHandler, 0)
	if busHandlers, ok := bus.streamHandlers[topic]; ok {
		handlers = append(handlers, busHandlers...)
	}

	return handlers
}

func (bus *EventBus) getPreprocessors(topic string) []idTopicProcessor {
	bus.busLock.RLock()
	defer bus.busLock.RUnlock()
	processors := make([]idTopicProcessor, 0)
	if busProcessors, ok := bus.preprocessors[topic]; ok {
		processors = append(processors, busProcessors...)
	}

	return processors
}
