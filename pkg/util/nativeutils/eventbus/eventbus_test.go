package eventbus

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

//*****************
// EVENTBUS TESTS
//*****************
func TestNewEventBus(t *testing.T) {
	eb := New()
	assert.NotNil(t, eb)
}

//*******************
// PREPROCESSOR TESTS
//*******************
func TestProcessor(t *testing.T) {
	topic := "testTopic"
	bus := New()

	resultChan := make(chan bytes.Buffer, 1)
	collector := NewSimpleCollector(resultChan, nil)

	ids := bus.Register(topic, NewAdder("pippo"), NewAdder("pippo"))
	NewTopicListener(bus, collector, topic, ChannelType)

	expected := *(bytes.NewBufferString("pippopippo"))
	bus.Publish(topic, bytes.NewBufferString(""))
	bus.Publish(topic, bytes.NewBufferString(""))

	result1 := <-resultChan
	result2 := <-resultChan
	assert.Equal(t, expected, result1)
	assert.Equal(t, expected, result2)

	// testing RemoveProcessor
	bus.RemoveProcessor(topic, ids[0])

	expected = *(bytes.NewBufferString("pippo"))
	bus.Publish(topic, bytes.NewBufferString(""))
	res := <-resultChan
	assert.Equal(t, expected, res)

	// removing the same preprocessor does not yield any different result
	bus.RemoveProcessor(topic, ids[0])
	bus.Publish(topic, bytes.NewBufferString(""))
	res = <-resultChan
	assert.Equal(t, expected, res)

	// adding a preprocessor
	expected = *(bytes.NewBufferString("pippopappo"))
	otherID := bus.Register(topic, NewAdder("pappo"))
	assert.Equal(t, 1, len(otherID))
	bus.Publish(topic, bytes.NewBufferString(""))
	res = <-resultChan
	assert.Equal(t, expected, res)

	// removing another
	expected = *(bytes.NewBufferString("pappo"))
	bus.RemoveProcessor(topic, ids[1])
	bus.Publish(topic, bytes.NewBufferString(""))
	res = <-resultChan
	assert.Equal(t, expected, res)
}

//******************
// SUBSCRIBER TESTS
//******************
func TestListenerMap(t *testing.T) {
	lm := newListenerMap()
	_, ss := CreateGossipStreamer()
	listener := NewStreamListener(ss)
	lm.Store("pippo", listener)

	listeners := lm.Load("pippo")
	assert.Equal(t, 1, len(listeners))
	assert.Equal(t, listener, listeners[0].Listener)
}

func TestSubscribe(t *testing.T) {
	eb := New()
	myChan := make(chan bytes.Buffer, 10)
	cl := NewChanListener(myChan)
	assert.NotNil(t, eb.Subscribe("whateverTopic", cl))
}

func TestUnsubscribe(t *testing.T) {
	eb, myChan, id := newEB(t)
	eb.Unsubscribe("whateverTopic", id)
	eb.Publish("whateverTopic", bytes.NewBufferString("whatever2"))

	select {
	case <-myChan:
		assert.FailNow(t, "We should have not received message")
	case <-time.After(50 * time.Millisecond):
		// success
	}
}

//*********************
// TOPIC LISTENER TESTS
//*********************
func TestLameSubscriber(t *testing.T) {
	bus := New()
	resultChan := make(chan bytes.Buffer, 1)

	collector := NewSimpleCollector(resultChan, nil)
	tbuf := ranbuf()

	sub := NewTopicListener(bus, collector, "pippo", ChannelType)

	bus.Publish("pippo", tbuf)
	bus.Publish("pippo", tbuf)

	assert.Equal(t, <-resultChan, *tbuf)
	assert.Equal(t, <-resultChan, *tbuf)

	sub.Quit()
	bus.Publish("pippo", tbuf)

	select {
	case <-resultChan:
		assert.FailNow(t, "unexpected message published after quitting a topic listener")
	case <-time.After(50 * time.Millisecond):
		//
	}
}

func TestStreamer(t *testing.T) {
	topic := string(topics.Gossip)
	bus, streamer := CreateFrameStreamer(topic)
	bus.Publish(topic, bytes.NewBufferString("pluto"))

	packet, err := streamer.(*SimpleStreamer).Read()
	if !assert.NoError(t, err) {
		assert.FailNow(t, "error in reading from the subscribed stream")
	}

	assert.Equal(t, "pluto", string(packet))
}

//******************
// MULTICASTER TESTS
//******************
func TestDefaultListener(t *testing.T) {
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

	eb.Publish("pippo", bytes.NewBufferString("pluto"))
	msg := <-msgChan
	assert.Equal(t, "pippo", msg.topic)
	assert.Equal(t, []byte("pluto"), msg.buf.Bytes())

	eb.Publish("paperino", bytes.NewBufferString("pluto"))
	msg = <-msgChan
	assert.Equal(t, "paperino", msg.topic)
	assert.Equal(t, []byte("pluto"), msg.buf.Bytes())

	eb.Publish("test", bytes.NewBufferString("pluto"))
	select {
	case <-msgChan:
		t.FailNow()
	case <-time.After(100 * time.Millisecond):
		//all good
	}
}

//****************
// SETUP FUNCTIONS
//****************
func newEB(t *testing.T) (*EventBus, chan bytes.Buffer, uint32) {
	eb := New()
	myChan := make(chan bytes.Buffer, 10)
	cl := NewChanListener(myChan)
	id := eb.Subscribe("whateverTopic", cl)
	assert.NotNil(t, id)
	b := bytes.NewBufferString("whatever")
	eb.Publish("whateverTopic", b)

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
	sl := NewStreamListener(&mockWriteCloser{})
	_ = eb.Subscribe(topic, sl)

	// Put something on ring buffer
	val := new(bytes.Buffer)
	val.Write([]byte{0})
	eb.Publish(topic, val)
	// Wait for event to be handled
	// NB: 'Writer' must return error to force consumer termination
	time.Sleep(100 * time.Millisecond)

	l := eb.listeners.Load(topic)
	for _, listener := range l {
		if streamer, ok := listener.Listener.(*StreamListener); ok {
			if !assert.True(t, streamer.ringbuffer.Closed()) {
				assert.FailNow(t, "ringbuffer not closed")
			}
			return
		}
	}
	assert.FailNow(t, "stream listener not found")
}

func ranbuf() *bytes.Buffer {
	tbytes, _ := crypto.RandEntropy(32)
	return bytes.NewBuffer(tbytes)
}

type mockWriteCloser struct {
}

func (m *mockWriteCloser) Write(data []byte) (int, error) {
	return 0, errors.New("failed")
}

func (m *mockWriteCloser) Close() error {
	return nil
}
