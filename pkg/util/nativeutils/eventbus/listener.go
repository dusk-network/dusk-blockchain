// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package eventbus

import (
	"crypto/rand"
	"errors"
	"math/big"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/hashset"
)

// Listener publishes a byte array that subscribers of the EventBus can use.
type Listener interface {
	// Notify a listener of a new message.
	Notify(message.Message) error
	// Close the listener.
	Close()
}

// CallbackListener subscribes using callbacks.
type CallbackListener struct {
	callback func(message.Message)
	safe     bool
}

// Notify the copy of a message as a parameter to a callback.
func (c *CallbackListener) Notify(m message.Message) error {
	if !c.safe {
		go c.callback(m)
		return nil
	}

	clone, err := message.Clone(m)
	if err != nil {
		log.WithError(err).Error("CallbackListener, failed to clone message")
		return err
	}

	go c.callback(clone)
	return nil
}

// NewSafeCallbackListener creates a callback based dispatcher.
func NewSafeCallbackListener(callback func(message.Message)) Listener {
	return &CallbackListener{callback, true}
}

// NewCallbackListener creates a callback based dispatcher.
func NewCallbackListener(callback func(message.Message)) Listener {
	return &CallbackListener{callback, false}
}

// Close as part of the Listener method.
func (c *CallbackListener) Close() {
}

var ringBufferLength = 2000

// StreamListener uses a ring buffer to dispatch messages. It is inherently
// thread-safe.
type StreamListener struct {
	ringbuffer *ring.Buffer
}

// NewStreamListener creates a new StreamListener.
func NewStreamListener(w ring.Writer) Listener {
	// Each StreamListener uses its own ringBuffer to collect topic events
	// Multiple-producers single-consumer approach utilizing a ringBuffer.
	ringBuf := ring.NewBuffer(ringBufferLength)
	sh := &StreamListener{ringBuf}

	// single-consumer
	_ = ring.NewConsumer(ringBuf, Consume, w, true)
	return sh
}

func NewStreamListenerWithLen(w ring.Writer, bufLen int) Listener {
	// Each StreamListener uses its own ringBuffer to collect topic events
	// Multiple-producers single-consumer approach utilizing a ringBuffer.
	ringBuf := ring.NewBuffer(bufLen)
	sh := &StreamListener{ringBuf}

	// single-consumer
	_ = ring.NewConsumer(ringBuf, Consume, w, true)
	return sh
}

// Notify puts a message to the Listener's ringbuffer. It uses a goroutine so
// to not block while the item is put in the ringbuffer.
func (s *StreamListener) Notify(m message.Message) error {
	// writing on the ringbuffer happens asynchronously
	go func() {
		buf := m.Payload().(message.SafeBuffer)

		e := ring.Elem{
			Data:     buf.Bytes(),
			Header:   m.Header(),
			Priority: 0,
		}

		if !s.ringbuffer.Put(e) {
			err := errors.New("ringbuffer is closed")
			logEB.WithField("queue", "ringbuffer").WithError(err).Warnln("ringbuffer closed")
		}
	}()

	return nil
}

// Close the internal ringbuffer.
func (s *StreamListener) Close() {
	if s.ringbuffer != nil {
		s.ringbuffer.Close()
	}
}

// Consume an item by writing it to the specified WriteCloser. This is used in the StreamListener creation.
func Consume(elems []ring.Elem, w ring.Writer) bool {
	for _, e := range elems {
		if _, err := w.Write(e.Data, e.Header, e.Priority); err != nil {
			logEB.WithField("queue", "ringbuffer").WithError(err).Warnln("error in writing to WriteCloser")
			return false
		}
	}

	return true
}

// ChanListener dispatches a message using a channel.
type ChanListener struct {
	messageChannel chan<- message.Message
	safe           bool
}

// NewChanListener creates a channel based dispatcher. Although the message is
// passed by value, this is not enough to enforce thread-safety when the
// listener tries to read/change slices or arrays carried by the message.
func NewChanListener(msgChan chan<- message.Message) Listener {
	return &ChanListener{msgChan, false}
}

// NewSafeChanListener creates a channel based dispatcher which is thread-safe.
func NewSafeChanListener(msgChan chan<- message.Message) Listener {
	return &ChanListener{msgChan, true}
}

// Notify sends a message to the internal dispatcher channel. It forwards the
// message if the listener is unsafe. Otherwise, it forwards a message clone.
func (c *ChanListener) Notify(m message.Message) error {
	if !c.safe {
		return forward(c.messageChannel, m)
	}

	clone, err := message.Clone(m)
	if err != nil {
		log.WithError(err).Error("ChanListener, failed to clone message")
		return err
	}

	return forward(c.messageChannel, clone)
}

// forward avoids code duplication in the ChanListener method.
func forward(msgChan chan<- message.Message, msg message.Message) error {
	select {
	case msgChan <- msg:
	default:
		return errors.New("message channel buffer is full")
	}

	return nil
}

// Close has no effect.
func (c *ChanListener) Close() {
}

// multilistener does not implement the Listener interface itself since the topic and
// the message category will likely differ. It delegates to the Notify method
// specified by the internal listener.
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

func (m *multiListener) Forward(topic topics.Topic, msg message.Message) (errorList []error) {
	m.RLock()
	defer m.RUnlock()

	if !m.Has([]byte{byte(topic)}) {
		return errorList
	}

	for _, dispatcher := range m.dispatchers {
		if err := dispatcher.Notify(msg); err != nil {
			logEB.WithError(err).WithField("type", "multilistener").Warnln("notifying subscriber failed")
			errorList = append(errorList, err)
		}
	}

	return errorList
}

func (m *multiListener) Store(value Listener) uint32 {
	// #654
	nBig, err := rand.Int(rand.Reader, big.NewInt(32))
	if err != nil {
		panic(err)
	}

	n := nBig.Int64()

	h := idListener{
		Listener: value,
		id:       uint32(n),
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
