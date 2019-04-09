package consensus

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func TestSECOperations(t *testing.T) {
	sec := NewStepEventCollector()
	ev1 := &MockEvent{"one"}
	ev2 := &MockEvent{"two"}
	ev3 := &MockEvent{"one"}

	// checking if the length of the array of step is consistent
	require.Equal(t, 1, sec.Store(ev1, "1"))
	require.Equal(t, 1, sec.Store(ev1, "1"))
	require.Equal(t, 1, sec.Store(ev1, "2"))
	require.Equal(t, 2, sec.Store(ev2, "2"))
	require.Equal(t, 2, sec.Store(ev3, "2"))

	sec.Clear()
	require.Equal(t, 1, sec.Store(ev1, "1"))
	require.Equal(t, 1, sec.Store(ev1, "1"))
	require.Equal(t, 1, sec.Store(ev1, "2"))
	require.Equal(t, 2, sec.Store(ev2, "2"))
	require.Equal(t, 2, sec.Store(ev3, "2"))
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
