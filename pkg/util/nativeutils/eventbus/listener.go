package eventbus

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/hashset"
)

// Listener publishes a byte array that subscribers of the EventBus can use
type Listener interface {
	// Notify a listener of a new message
	Notify(message.Message) error
	// Close the listener
	Close()
}

// CallbackListener subscribes using callbacks
type CallbackListener struct {
	callback func(message.Message) error
}

// Notify the copy of a message as a parameter to a callback
func (c *CallbackListener) Notify(m message.Message) error {
	return c.callback(m)
}

// NewCallbackListener creates a callback based dispatcher
func NewCallbackListener(callback func(message.Message) error) Listener {
	return &CallbackListener{callback}
}

// Close as part of the Listener method
func (c *CallbackListener) Close() {
}

var ringBufferLength = 2000

// StreamListener uses a ring buffer to dispatch messages
type StreamListener struct {
	ringbuffer *ring.Buffer
}

// NewStreamListener creates a new StreamListener
func NewStreamListener(w io.WriteCloser) Listener {
	// Each StreamListener uses its own ringBuffer to collect topic events
	// Multiple-producers single-consumer approach utilizing a ringBuffer
	ringBuf := ring.NewBuffer(ringBufferLength)
	sh := &StreamListener{ringBuf}

	// single-consumer
	_ = ring.NewConsumer(ringBuf, Consume, w)
	return sh
}

// Notify puts a message to the Listener's ringbuffer
func (s *StreamListener) Notify(m message.Message) error {
	if s.ringbuffer == nil {
		return errors.New("no ringbuffer specified")
	}

	// TODO: interface - the ring buffer should be able to handle interface
	// payloads rather than restricting solely to buffers
	// TODO: interface - This panics in case payload is not a buffer
	buf := m.Payload().(bytes.Buffer)
	s.ringbuffer.Put(buf.Bytes())
	return nil
}

// Close the internal ringbuffer
func (s *StreamListener) Close() {
	if s.ringbuffer != nil {
		s.ringbuffer.Close()
	}
}

// Consume an item by writing it to the specified WriteCloser. This is used in the StreamListener creation
func Consume(items [][]byte, w io.WriteCloser) bool {
	for _, data := range items {
		if _, err := w.Write(data); err != nil {
			logEB.WithField("queue", "ringbuffer").WithError(err).Warnln("error in writing to WriteCloser")
			return false
		}
	}

	return true
}

// ChanListener dispatches a message using a channel
type ChanListener struct {
	messageChannel chan<- message.Message
}

// NewChanListener creates a channel based dispatcher
func NewChanListener(msgChan chan<- message.Message) Listener {
	return &ChanListener{msgChan}
}

// Notify sends a message to the internal dispatcher channel
func (c *ChanListener) Notify(m message.Message) error {
	select {
	case c.messageChannel <- m:
	default:
		return errors.New("message channel buffer is full")
	}

	return nil
}

// Close has no effect
func (c *ChanListener) Close() {
}

// multilistener does not implement the Listener interface since the topic and
// the message category will likely differ
type multiListener struct {
	sync.RWMutex
	*hashset.Set
	dispatchers []idListener
}

func newMultiListener() *multiListener {
	return &multiListener{
		Set:         hashset.New(),
		dispatchers: make([]idListener, 0),
	}
}

func (m *multiListener) Add(topic topics.Topic) {
	m.RWMutex.Lock()
	defer m.RWMutex.Unlock()
	m.Set.Add([]byte{byte(topic)})
}

func (m *multiListener) Forward(topic topics.Topic, msg message.Message) {
	m.RLock()
	defer m.RUnlock()
	if !m.Has([]byte{byte(topic)}) {
		return
	}

	for _, dispatcher := range m.dispatchers {
		if err := dispatcher.Notify(msg); err != nil {
			logEB.WithError(err).WithField("type", "multilistener").Warnln("notifying subscriber failed")
		}
	}
}

func (m *multiListener) Store(value Listener) uint32 {
	h := idListener{
		Listener: value,
		id:       rand.Uint32(),
	}
	m.Lock()
	defer m.Unlock()
	m.dispatchers = append(m.dispatchers, h)
	return h.id
}

func (m *multiListener) Delete(id uint32) bool {
	m.Lock()
	defer m.Unlock()

	for i, h := range m.dispatchers {
		if h.id == id {
			h.Close()
			m.dispatchers = append(
				m.dispatchers[:i],
				m.dispatchers[i+1:]...,
			)
			return true
		}
	}
	return false
}
