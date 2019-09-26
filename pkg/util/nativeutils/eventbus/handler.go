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

// Handler publishes a byte array that subscribers of the EventBus can use
type Handler interface {
	Publish(*bytes.Buffer) error
	Close()
}

type callbackHandler struct {
	callback func(*bytes.Buffer) error
}

func (c *callbackHandler) Publish(m *bytes.Buffer) error {
	mCopy := copyBuffer(m)
	return c.callback(mCopy)
}

func (c *callbackHandler) Close() {
}

type streamHandler struct {
	topic      string
	ringbuffer *ring.Buffer
}

func (s *streamHandler) Publish(m *bytes.Buffer) error {
	if s.ringbuffer == nil {
		return errors.New("no ringbuffer specified")
	}
	s.ringbuffer.Put(m.Bytes())
	return nil
}

func (s *streamHandler) Close() {
	if s.ringbuffer != nil {
		s.ringbuffer.Close()
	}
}

type channelHandler struct {
	messageChannel chan<- *bytes.Buffer
}

func (c *channelHandler) Publish(m *bytes.Buffer) error {
	mCopy := copyBuffer(m)
	select {
	case c.messageChannel <- mCopy:
	default:
		return errors.New("message channel buffer is full")
	}

	return nil
}

func (c *channelHandler) Close() {
}

type multiDispatcher struct {
	sync.RWMutex
	*hashset.Set
	dispatchers []idHandler
}

func newMultiDispatcher() *multiDispatcher {
	return &multiDispatcher{
		Set:         hashset.New(),
		dispatchers: make([]idHandler, 0),
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
			dispatcher.Publish(tpcMsg)
		}
		m.RUnlock()
	}
}

func (m *multiDispatcher) Store(value Handler) uint32 {
	h := idHandler{
		Handler: value,
		id:      rand.Uint32(),
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
