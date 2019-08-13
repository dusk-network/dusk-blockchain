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
