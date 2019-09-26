package wire_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLameSubscriber(t *testing.T) {
	bus := eventbus.New()
	resultChan := make(chan *bytes.Buffer, 1)
	collector := defaultMockCollector(resultChan, nil)
	tbuf := ranbuf()

	sub := eventbus.NewTopicListener(bus, collector, "pippo")
	go sub.Accept()

	// NOTE: in real life we would never reuse the same buffer as it would most certainly be mutated by the processor or the collector
	bus.Publish("pippo", tbuf)
	bus.Publish("pippo", tbuf)
	require.Equal(t, <-resultChan, tbuf)
	require.Equal(t, <-resultChan, tbuf)
}

func TestProcessor(t *testing.T) {
	topic := "testTopic"
	bus := eventbus.New()
	resultChan := make(chan *bytes.Buffer, 1)
	collector := defaultMockCollector(resultChan, nil)

	ids := bus.RegisterPreprocessor(topic, &pippoAdder{}, &pippoAdder{})
	go eventbus.NewTopicListener(bus, collector, topic).Accept()

	expected := bytes.NewBufferString("pippopippo")
	bus.Publish(topic, bytes.NewBufferString(""))
	bus.Publish(topic, bytes.NewBufferString(""))

	result1 := <-resultChan
	result2 := <-resultChan
	assert.Equal(t, expected, result1)
	assert.Equal(t, expected, result2)

	// testing RemoveProcessor
	bus.RemovePreprocessor(topic, ids[0])

	expected = bytes.NewBufferString("pippo")
	bus.Publish(topic, bytes.NewBufferString(""))
	res := <-resultChan
	assert.Equal(t, expected, res)

	// removing the same preprocessor does not yield any different result
	bus.RemovePreprocessor(topic, ids[0])
	bus.Publish(topic, bytes.NewBufferString(""))
	res = <-resultChan
	assert.Equal(t, expected, res)

	// adding a preprocessor
	expected = bytes.NewBufferString("pippopappo")
	otherId := bus.RegisterPreprocessor(topic, &pappoAdder{})
	assert.Equal(t, 1, len(otherId))
	bus.Publish(topic, bytes.NewBufferString(""))
	res = <-resultChan
	assert.Equal(t, expected, res)

	// removing another
	expected = bytes.NewBufferString("pappo")
	bus.RemovePreprocessor(topic, ids[1])
	bus.Publish(topic, bytes.NewBufferString(""))
	res = <-resultChan
	assert.Equal(t, expected, res)
}

func TestAddTopic(t *testing.T) {
	buf := bytes.NewBufferString("This is a test")
	topic := topics.Gossip
	newBuffer, err := wire.AddTopic(buf, topic)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x67, 0x6f, 0x73, 0x73, 0x69, 0x70, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x74, 0x65, 0x73, 0x74}, newBuffer.Bytes())
}

func TestQuit(t *testing.T) {
	bus := eventbus.New()
	sub := eventbus.NewTopicListener(bus, nil, "")
	go func() {
		time.Sleep(50 * time.Millisecond)
		bus.Publish(string(msg.QuitTopic), nil)
	}()
	sub.Accept()
	//after 50ms the Quit should kick in and unblock Accept()
}

type pippoAdder struct{}

func (p *pippoAdder) Process(buf *bytes.Buffer) (*bytes.Buffer, error) {
	buf.WriteString("pippo")
	return buf, nil
}

type pappoAdder struct{}

func (p *pappoAdder) Process(buf *bytes.Buffer) (*bytes.Buffer, error) {
	buf.WriteString("pappo")
	return buf, nil
}

type mockCollector struct {
	f func(*bytes.Buffer) error
}

func defaultMockCollector(rChan chan *bytes.Buffer, f func(*bytes.Buffer) error) *mockCollector {
	if f == nil {
		f = func(b *bytes.Buffer) error {
			rChan <- b
			return nil
		}
	}
	return &mockCollector{f}
}

func (m *mockCollector) Collect(b *bytes.Buffer) error { return m.f(b) }

func ranbuf() *bytes.Buffer {
	tbytes, _ := crypto.RandEntropy(32)
	return bytes.NewBuffer(tbytes)
}
