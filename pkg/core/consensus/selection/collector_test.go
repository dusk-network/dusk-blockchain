package selection

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func TestNoCollectionBeforeStart(t *testing.T) {
	// ev1 := &consensus.EventHeader{Round: 1, Step: 1}

}
func TestStopSelectorWithoutResult(t *testing.T) {
	selector := NewEventSelector(&MockPrioritizer{}, make(chan wire.Event, 3))
	go selector.PickBest()
	selector.EventChan <- &MockEvent{"one"}
	selector.EventChan <- &MockEvent{"two"}
	selector.EventChan <- &MockEvent{"three"}
	selector.StopChan <- false

	select {
	case <-selector.BestEventChan:
		assert.FailNow(t, "Selector should have not returned a value")
	case <-time.After(200 * time.Millisecond):
		// assert.Equal(t, &MockEvent{"one"}, selector.bestEvent)
		// success :)
	}
}

func TestStopSelectorWithResult(t *testing.T) {
	selector := NewEventSelector(&MockPrioritizer{}, make(chan wire.Event, 3))
	go selector.PickBest()
	selector.EventChan <- &MockEvent{"one"}
	selector.EventChan <- &MockEvent{"two"}
	selector.EventChan <- &MockEvent{"three"}
	selector.StopChan <- true

	select {
	case ev := <-selector.BestEventChan:
		assert.Equal(t, &MockEvent{"one"}, ev)
	case <-time.After(20 * time.Millisecond):
		assert.FailNow(t, "Selector should have returned a value")
	}
}

type MockPrioritizer struct{}

// Priority is a stupid function that returns always the first Event
func (mp *MockPrioritizer) Priority(f, s wire.Event) wire.Event {
	if f == nil {
		return s
	}
	return f
}

type MockEvent struct {
	field string
}

func (me *MockEvent) Equal(ev wire.Event) bool {
	return reflect.DeepEqual(me, ev)
}

func (me *MockEvent) Unmarshal(b *bytes.Buffer) error { return nil }

func (me *MockEvent) Sender() []byte {
	return []byte{}
}
