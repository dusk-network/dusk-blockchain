package wire

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
)

// Handler publishes a byte array that subscribers of the EventBus can use
type Handler interface {
	Publish(*bytes.Buffer) error
	Close()
	// TODO: get rid of the uint32
	ID() uint32
}

type callbackHandler struct {
	id       uint32
	callback func(*bytes.Buffer) error
}

func (c *callbackHandler) Publish(m *bytes.Buffer) error {
	mCopy := copyBuffer(m)
	return c.callback(mCopy)
}

func (c *callbackHandler) ID() uint32 {
	return c.id
}

func (c *callbackHandler) Close() {
}

type streamHandler struct {
	id         uint32
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

func (s *streamHandler) ID() uint32 {
	return s.id
}

func (s *streamHandler) Close() {
	if s.ringbuffer != nil {
		s.ringbuffer.Close()
	}
}

type channelHandler struct {
	id             uint32
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

func (c *channelHandler) ID() uint32 {
	return c.id
}

func (c *channelHandler) Close() {
}
