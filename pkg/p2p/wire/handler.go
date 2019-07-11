package wire

import (
	"bytes"
	"errors"
)

type Handler interface {
	Publish(*bytes.Buffer) error
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

type streamHandler struct {
	id       uint32
	exitChan chan struct{}
	topic    string
}

func (s *streamHandler) Publish(m *bytes.Buffer) error {
	return nil
}

func (s *streamHandler) ID() uint32 {
	return s.id
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
