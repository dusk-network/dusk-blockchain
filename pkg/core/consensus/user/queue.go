package user

import (
	"sync"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

// Queue holds all pending consensus messages
type Queue struct {
	*sync.Map
}

// Get will get all the messages for a certain round and step
func (q *Queue) Get(round uint64, step uint32) []*payload.MsgConsensus {
	m, ok := q.Load(round)
	if !ok {
		return nil
	}

	arr, ok := m.(*sync.Map).Load(step)
	if !ok {
		return nil
	}

	return arr.([]*payload.MsgConsensus)
}

// Put will put a message in the array for the designated round and step
func (q *Queue) Put(round uint64, step uint32, msg *payload.MsgConsensus) {
	m, ok := q.Load(round)
	if !ok {
		newMap := new(sync.Map)
		msgs := make([]*payload.MsgConsensus, 0)
		msgs = append(msgs, msg)
		newMap.Store(step, msgs)
		q.Store(round, newMap)
		return
	}

	arr, ok := m.(*sync.Map).Load(step)
	if !ok {
		msgs := make([]*payload.MsgConsensus, 0)
		msgs = append(msgs, msg)
		m.(*sync.Map).Store(step, msgs)
		return
	}

	msgs := arr.([]*payload.MsgConsensus)
	msgs = append(msgs, msg)
	m.(*sync.Map).Store(step, msgs)
}
