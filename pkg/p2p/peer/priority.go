package peer

import (
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"go.uber.org/atomic"
	"sync"
)

// PriorityQueue is a kind of FIFO queue that aims to fairly distribute work but
// favour priority work when it is dominating the message queue.
//
// It exploits the FIFO queue semantics of buffered channels and uses atomics to
// track the buffer utilization.
//
// Using modulo on the factor value, it will select a priority item when the factor is even, and
// in the next, it will by default then select a non-priority
type PriorityQueue struct {
	sync.Mutex
	factor                uint32
	Priority, Nonpriority chan []byte
	Plen, Nlen            *atomic.Uint32
}

// NewPriorityQueue creates a new priority queue for messages
//
// - size sets the size of each of the queue buckets
//
// - factor sets a number that by the sum of items in the queue a priority item
//   is always selected, if there is one remaining, it acts like an interleaving
//   factor, every second, every third, or so, as defined, will consume an
//   available priority item.
//
func NewPriorityQueue(size int, factor uint32) (pq *PriorityQueue) {
	pq = &PriorityQueue{
		Mutex:       sync.Mutex{},
		factor:      factor,
		Priority:    make(chan []byte, size),
		Nonpriority: make(chan []byte, size),
		Plen:        atomic.NewUint32(0),
		Nlen:        atomic.NewUint32(0),
	}
	return
}

// Push a message on the queue, checking if it is a priority message type, and
// putting it into the channel relevant to the priority.
//
// If the channel buffer is full, this function will block, and it is recommended
// to increase the size specified in the constructor. If the buffer usage is too
// high or peaky, then there may be bottlenecks in the processing phase.
func (pq *PriorityQueue) Push(msg []byte) {
	if len(msg) > 0 {
		for _, v := range topics.Priority {
			if msg[0] == byte(v) {
				// this is priority message
				select {
				case pq.Priority <- msg:
				default:
					// if the above is blocking, buffer needs to be increased, thus the panic
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
			// if the above is blocking, buffer needs to be increased, thus panic
			panic("priority queue is breached, buffer needs to be increased in size")
		}
		// once channel loads increment atomic counter
		pq.Nlen.Inc()
	} // if message was empty for now silently no-op
}

// Pop a message from the PriorityQueue.
//
// The queue empties by the following rules:
//
// - If there is none, return nil
// - If there is some in one and none in the other, empty the one with some
// - If there is equal or more priority items than non-priority, process the priority
//   items
// - If there is more non-priority items than priority, process the non-priority
// - It always chooses priority every [factor-th] message, so non-priority messages
//   cannot flood it, and vice versa, every
//
// This gives favour to priority messages but aims to empty the queue equally, so
// mostly it interleaves the jobs. This scheme does not introduce a vulnerability
// where flooding messages of either type will deny processing to the other, as
// the queue pops non-priority items as soon as there is more of them than
// non-priority.
func (pq *PriorityQueue) Pop() (msg []byte) {
	P, N := pq.Plen.Load(), pq.Nlen.Load()
	switch {
	case P == 0 && N == 0:
		// if there is no messages, return empty message
		return
	case P > 0 && (N < P || P%pq.factor == 0):
		// If there is more priority messages, or if the sum of number of messages is a
		// factor of the set priority and a priority message exists send priority.
		msg = <-pq.Priority
		pq.Plen.Dec()
	case N > 0:
		// if the previous cases did not fall through and a non-priority message exists,
		// send it
		msg = <-pq.Nonpriority
		pq.Nlen.Dec()
	}
	return
}
