package wire

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

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

func TestLameSubscriber(t *testing.T) {
	bus := New()
	resultChan := make(chan *bytes.Buffer, 1)
	collector := defaultMockCollector(resultChan, nil)
	tbuf := ranbuf()

	sub := NewEventSubscriber(bus, collector, "pippo")
	go sub.Accept()

	bus.Publish("pippo", tbuf)
	bus.Publish("pippo", tbuf)
	require.Equal(t, <-resultChan, tbuf)
	require.Equal(t, <-resultChan, tbuf)
}

func TestQuit(t *testing.T) {
	bus := New()
	sub := NewEventSubscriber(bus, nil, "")
	go func() {
		time.Sleep(50 * time.Millisecond)
		bus.Publish(string(msg.QuitTopic), nil)
	}()
	sub.Accept()
	//after 50ms the Quit should kick in and unblock Accept()
}

func TestStopSelectorWithResult(t *testing.T) {
	selector := NewEventSelector(&MockPrioritizer{})
	go selector.PickBest()
	selector.EventChan <- &MockEvent{"one"}
	selector.EventChan <- &MockEvent{"two"}
	selector.EventChan <- &MockEvent{"three"}
	selector.StopChan <- true

	select {
	case ev := <-selector.BestEventChan:
		assert.Equal(t, &MockEvent{"one"}, ev)
	case <-time.After(20):
		assert.FailNow(t, "Selector should have returned a value")
	}
}
func TestStopSelectorWithoutResult(t *testing.T) {
	selector := NewEventSelector(&MockPrioritizer{})
	go selector.PickBest()
	selector.EventChan <- &MockEvent{"one"}
	selector.EventChan <- &MockEvent{"two"}
	selector.EventChan <- &MockEvent{"three"}
	selector.StopChan <- false

	select {
	case <-selector.BestEventChan:
		assert.FailNow(t, "Selector should have not returned a value")
	case <-time.After(20):
		assert.Equal(t, &MockEvent{"one"}, selector.bestEvent)
	}
}
func TestSECOperations(t *testing.T) {
	sec := &StepEventCollector{}
	ev1 := &MockEvent{"one"}
	ev2 := &MockEvent{"two"}
	ev3 := &MockEvent{"one"}

	// checking if the length of the array of step is consistent
	require.Equal(t, 1, sec.Store(ev1, 1))
	require.Equal(t, 1, sec.Store(ev1, 1))
	require.Equal(t, 1, sec.Store(ev1, 2))
	require.Equal(t, 2, sec.Store(ev2, 2))
	require.Equal(t, 2, sec.Store(ev3, 2))

	sec.Clear()
	require.Equal(t, 1, sec.Store(ev1, 1))
	require.Equal(t, 1, sec.Store(ev1, 1))
	require.Equal(t, 1, sec.Store(ev1, 2))
	require.Equal(t, 2, sec.Store(ev2, 2))
	require.Equal(t, 2, sec.Store(ev3, 2))
}

type MockPrioritizer struct{}

// Priority is a stupid function that returns always the first Event
func (mp *MockPrioritizer) Priority(f, s Event) Event {
	if f == nil {
		return s
	}
	return f
}

type MockEvent struct {
	field string
}

func (me *MockEvent) Equal(ev Event) bool {
	return reflect.DeepEqual(me, ev)
}

func (me *MockEvent) Unmarshal(b *bytes.Buffer) error { return nil }

func (me *MockEvent) Sender() []byte {
	return []byte{}
}
