package eventbus

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/hashset"
	log "github.com/sirupsen/logrus"
)

// Listener publishes a byte array that subscribers of the EventBus can use
type Listener interface {
	// TODO: Publish should pass the buffer by value rather than by reference
	// to prevent ad hoc copying the buffer
	Notify(bytes.Buffer) error
	Close()
}

// CallbackListener subscribes using callbacks
type CallbackListener struct {
	callback func(bytes.Buffer) error
}

// Publish the copy of a message as a parameter to a callback
func (c *CallbackListener) Notify(m bytes.Buffer) error {
	return c.callback(m)
}

// NewCallbackListener creates a callback based dispatcher
func NewCallbackListener(callback func(bytes.Buffer) error) Listener {
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
func (s *StreamListener) Notify(m bytes.Buffer) error {
	if s.ringbuffer == nil {
		return errors.New("no ringbuffer specified")
	}
	s.ringbuffer.Put(m.Bytes())
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
	messageChannel chan<- bytes.Buffer
}

// NewChanListener creates a channel based dispatcher
func NewChanListener(msgChan chan<- bytes.Buffer) Listener {
	return &ChanListener{msgChan}
}

// Publish sends a message to the internal dispatcher channel
func (c *ChanListener) Notify(m bytes.Buffer) error {
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

func (m *multiListener) Notify(topic string, r bytes.Buffer) {
	if m.Has([]byte(topic)) {
		tpc := topics.StringToTopic(topic)
		// creating a new Buffer carrying also the topic
		tpcMsg := new(bytes.Buffer)
		topics.Write(tpcMsg, tpc)

		if _, err := r.WriteTo(tpcMsg); err != nil {
			log.WithField("topic", topic).WithError(err).Warnln("error in writing topic to a multi-dispatched packet")
			return
		}

		m.RLock()
		for _, dispatcher := range m.dispatchers {
			dispatcher.Notify(*tpcMsg)
		}
		m.RUnlock()
	}
}

func (m *multiListener) Store(value Listener) uint32 {
	h := idListener{
		Listener: value,
		id:       rand.Uint32(),
	}
	m.Lock()
	m.dispatchers = append(m.dispatchers, h)
	m.Unlock()
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
