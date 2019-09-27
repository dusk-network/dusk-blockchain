package eventbus

import (
	"bytes"
	"errors"
	"math/rand"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/hashset"
	log "github.com/sirupsen/logrus"
)

// Dispatcher publishes a byte array that subscribers of the EventBus can use
type Dispatcher interface {
	// TODO: Publish should pass the buffer by value rather than by reference
	// to prevent ad hoc copying the buffer
	Publish(bytes.Buffer) error
	Close()
}

type callbackDispatcher struct {
	callback func(bytes.Buffer) error
}

func (c *callbackDispatcher) Publish(m bytes.Buffer) error {
	return c.callback(m)
}

func (c *callbackDispatcher) Close() {
}

type streamDispatcher struct {
	topic      string
	ringbuffer *ring.Buffer
}

func (s *streamDispatcher) Publish(m bytes.Buffer) error {
	if s.ringbuffer == nil {
		return errors.New("no ringbuffer specified")
	}
	s.ringbuffer.Put(m.Bytes())
	return nil
}

func (s *streamDispatcher) Close() {
	if s.ringbuffer != nil {
		s.ringbuffer.Close()
	}
}

type channelDispatcher struct {
	messageChannel chan<- bytes.Buffer
}

func (c *channelDispatcher) Publish(m bytes.Buffer) error {
	select {
	case c.messageChannel <- m:
	default:
		return errors.New("message channel buffer is full")
	}

	return nil
}

func (c *channelDispatcher) Close() {
}

type multiDispatcher struct {
	sync.RWMutex
	*hashset.Set
	dispatchers []idDispatcher
}

func newMultiDispatcher() *multiDispatcher {
	return &multiDispatcher{
		Set:         hashset.New(),
		dispatchers: make([]idDispatcher, 0),
	}
}

func (m *multiDispatcher) Publish(topic string, r bytes.Buffer) {
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
			dispatcher.Publish(*tpcMsg)
		}
		m.RUnlock()
	}
}

func (m *multiDispatcher) Store(value Dispatcher) uint32 {
	h := idDispatcher{
		Dispatcher: value,
		id:         rand.Uint32(),
	}
	m.Lock()
	m.dispatchers = append(m.dispatchers, h)
	m.Unlock()
	return h.id
}

func (m *multiDispatcher) Delete(id uint32) bool {
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
