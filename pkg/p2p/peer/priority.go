package peer

import (
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"go.uber.org/atomic"
	"sync"
)

type PriorityQueue struct {
	sync.Mutex
	Priority, Nonpriority chan []byte
	Plen, Nlen            *atomic.Uint32
}

func NewPriorityQueue(size int) (pq *PriorityQueue) {
	pq = &PriorityQueue{
		Mutex:       sync.Mutex{},
		Priority:    make(chan []byte, size*2/3),
		Nonpriority: make(chan []byte, size/3),
		Plen:        atomic.NewUint32(0),
		Nlen:        atomic.NewUint32(0),
	}
	return
}

// Push a message on the queue, checking if it is a priority message type, and
// putting it into the channel relevant to the priority.
//
// If the channel buffer is full, this function will block, and it is
// recommended to increase the size specified in the constructor
func (pq *PriorityQueue) Push(msg []byte) {
	if len(msg) > 0 {
		for _, v := range topics.Priority {
			if msg[0] == byte(v) {
				// this is priority message
				select {
				case pq.Priority <- msg:
				default:
					// if the above is blocking, buffer needs to be increased
					panic("priority queue is breached, buffer needs to be increased in size")
				}
				// once channel loads increment atomic counter
				pq.Plen.Inc()
				return
			}
		}
		// this is not a priority message
		select {
		case pq.Nonpriority <- msg:
		default:
			// if the above is blocking, buffer needs to be increased
			panic("priority queue is breached, buffer needs to be increased in size")
		}
		// once channel loads increment atomic counter
		pq.Nlen.Inc()
	} // if message was empty for now silently no-op
}

// Pop a message from the PriorityQueue.
//
// - If the queue is empty, nil is returned.
// - If there is more than double the number of priority items than non-priority, return the priority
// - Otherwise return the non-priority
//
//
func (pq *PriorityQueue) Pop() (msg []byte) {
	P, N := pq.Plen.Load()+1, pq.Nlen.Load()+1
	if P == 1 && N == 1 {
		// queue is empty
		return
	}
	if P/N >= 2 {
		// If there is more than double priority items than non-priority
		// process the priority item
		msg = <-pq.Priority
		// decrease counter on priority queue
		pq.Plen.Dec()
	} else {
		// otherwise process non-priority item
		msg = <-pq.Nonpriority
		// decrease counter on non-priority queue
		pq.Nlen.Dec()
	}
	return
}
