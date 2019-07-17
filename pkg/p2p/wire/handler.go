package wire

import (
	"bytes"
	"errors"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/container/ring"
)

type Handler interface {
	Publish(*bytes.Buffer) error
	ID() uint32
	GetBuffer() *ring.Buffer
	Close()
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

func (c *callbackHandler) GetBuffer() *ring.Buffer {
	return nil
}

func (c *callbackHandler) Close() {
}

type streamHandler struct {
	id         uint32
	exitChan   chan struct{}
	topic      string
	ringbuffer *ring.Buffer
}

func (s *streamHandler) GetBuffer() *ring.Buffer {
	return s.ringbuffer
}

func (s *streamHandler) Publish(m *bytes.Buffer) error {
	return nil
}

func (s *streamHandler) ID() uint32 {
	return s.id
}

func (c *streamHandler) Close() {
	if c.ringbuffer != nil {
		c.ringbuffer.Close()
	}
}

type channelHandler struct {
	id             uint32
	messageChannel chan<- *bytes.Buffer
}

func (s *channelHandler) GetBuffer() *ring.Buffer {
	return nil
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
