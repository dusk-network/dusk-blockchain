package user

import (
	"sync"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// Queue holds pending consensus messages
type Queue struct {
	*sync.Map
}

// Get will get all the messages for a certain round and step
func (q *Queue) Get(round uint64, step uint8) []wire.Payload {
	m, ok := q.Load(round)
	if !ok {
		return nil
	}

	arr, ok := m.(*sync.Map).Load(step)
	if !ok {
		return nil
	}

	return arr.([]wire.Payload)
}

// Put will put a message in the array for the designated round and step
func (q *Queue) Put(round uint64, step uint8, msg wire.Payload) {
	m, ok := q.Load(round)
	if !ok {
		newMap := new(sync.Map)
		msgs := make([]wire.Payload, 0)
		msgs = append(msgs, msg)
		newMap.Store(step, msgs)
		q.Store(round, newMap)
		return
	}

	arr, ok := m.(*sync.Map).Load(step)
	if !ok {
		msgs := make([]wire.Payload, 0)
		msgs = append(msgs, msg)
		m.(*sync.Map).Store(step, msgs)
		return
	}

	msgs := arr.([]wire.Payload)
	msgs = append(msgs, msg)
	m.(*sync.Map).Store(step, msgs)
}
