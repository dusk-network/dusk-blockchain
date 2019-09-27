package eventbus

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/stretchr/testify/assert"
)

func TestNewEventBus(t *testing.T) {
	eb := New()
	assert.NotNil(t, eb)
}

func TestSubscribe(t *testing.T) {
	eb := New()
	myChan := make(chan bytes.Buffer, 10)
	assert.NotNil(t, eb.Subscribe("whateverTopic", myChan))
}

func TestPublish(t *testing.T) {
	newEB(t)
}

func TestUnsubscribe(t *testing.T) {
	eb, myChan, id := newEB(t)
	eb.Unsubscribe("whateverTopic", id)

	eb.Publish("whateverTopic", *bytes.NewBufferString("whatever2"))
	select {
	case <-myChan:
		assert.FailNow(t, "We should have not received message")
	case <-time.After(50 * time.Millisecond):
		// success
	}
}

func TestDefaultDispatcher(t *testing.T) {
	eb := New()
	msgChan := make(chan struct {
		topic string
		buf   bytes.Buffer
	})

	cb := func(r bytes.Buffer) error {
		tpc, _ := topics.Extract(&r)

		msgChan <- struct {
			topic string
			buf   bytes.Buffer
		}{string(tpc), r}
		return nil
	}

	eb.AddDefaultTopic("pippo")
	eb.AddDefaultTopic("paperino")
	eb.SubscribeDefault(cb)

	eb.Publish("pippo", *bytes.NewBufferString("pluto"))
	msg := <-msgChan
	assert.Equal(t, "pippo", msg.topic)
	assert.Equal(t, []byte("pluto"), msg.buf.Bytes())

	eb.Publish("paperino", *bytes.NewBufferString("pluto"))
	msg = <-msgChan
	assert.Equal(t, "paperino", msg.topic)
	assert.Equal(t, []byte("pluto"), msg.buf.Bytes())

	eb.Publish("test", *bytes.NewBufferString("pluto"))
	select {
	case <-msgChan:
		t.FailNow()
	case <-time.After(100 * time.Millisecond):
		//all good
	}
}

func newEB(t *testing.T) (*EventBus, chan bytes.Buffer, uint32) {
	eb := New()
	myChan := make(chan bytes.Buffer, 10)
	id := eb.Subscribe("whateverTopic", myChan)
	assert.NotNil(t, id)

	eb.Publish("whateverTopic", *bytes.NewBufferString("whatever"))

	select {
	case received := <-myChan:
		assert.Equal(t, "whatever", received.String())
	case <-time.After(50 * time.Millisecond):
		assert.FailNow(t, "We should have received a message by now")
	}

	return eb, myChan, id
}

// Test that a streaming goroutine is killed when the exit signal is sent
func TestExitChan(t *testing.T) {

	eb := New()
	topic := "foo"
	_ = eb.SubscribeStream(topic, &mockWriteCloser{})
	// Put something on ring buffer
	val := new(bytes.Buffer)
	val.Write([]byte{0})
	eb.Stream(topic, *val)
	// Wait for event to be handled
	// NB: 'Writer' must return error to force consumer termination
	time.Sleep(100 * time.Millisecond)

	var closed bool
	dispatchers := eb.streamDispatchers.Load(topic)
	for _, dispatcher := range dispatchers {
		sh := dispatcher.(*streamDispatcher)
		closed = sh.ringbuffer.Closed()
	}

	assert.True(t, closed)
}

type mockWriteCloser struct {
}

func (m *mockWriteCloser) Write(data []byte) (int, error) {
	return 0, errors.New("failed")
}

func (m *mockWriteCloser) Close() error {
	return nil
}
