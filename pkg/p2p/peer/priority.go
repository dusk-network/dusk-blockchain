package peer

import (
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"go.uber.org/atomic"
	"sync"
)

// Priority topics should be handled in an even proportion to ensure network
// critical messages are processed in a timely fashion
var Priority = []topics.Topic{
	// consensus messages need to be responded to quickly to improve network convergence
	topics.Candidate,
	topics.NewBlock,
	topics.Reduction,
	topics.Agreement,
	// these will also be priority because they are anyway hard to spam
	topics.Kadcast,
	topics.KadcastPoint,
	// if a new block has arrived, this is a priority event as it needs
	// to be attached to the chain as soon as possible in order to mine
	topics.AcceptedBlock,
}

// PriorityQueue is a kind of FIFO queue that aims to fairly distribute work but
// favour priority work when it is dominating the message queue.
//
// It exploits the FIFO queue semantics of buffered channels and uses atomics to
// track the buffer utilization.
//
// The queue's primary mode of operation is to consume items from the subqueue
// whose buffer is more full, but in addition it forces selection of the subqueue
// every time the number of items is a whole product of the `factor` field below
// which essentially makes sure even if one queue is stacking up faster than the
// other, that they don't get too stale.
type PriorityQueue struct {
	sync.RWMutex
	size                  int
	factor                uint32
	Priority, Nonpriority chan []byte
	Plen, Nlen            *atomic.Uint32
}

// NewPriorityQueue creates a new priority queue for messages
//
// - size sets the size of each of the queue buckets
//
// - factor sets a number which is used with the queue sizes and every
//   time the priority queue is one of these sizes it will be picked,
//   and likewise if the non-priority queue is this size, it will be
//   picked if this same rule didn't pick the priority item already.
func NewPriorityQueue(size int, factor uint32) (pq *PriorityQueue) {
	pq = &PriorityQueue{
		RWMutex:     sync.RWMutex{},
		size:        size,
		factor:      factor,
		Priority:    make(chan []byte, size),
		Nonpriority: make(chan []byte, size),
		Plen:        atomic.NewUint32(0),
		Nlen:        atomic.NewUint32(0),
	}
	return
}

// SetFactor changes the factor by which a given sub-queue is forced to be popped
// from
func (pq *PriorityQueue) SetFactor(factor uint32) {
	// the full lock is done here because this changes what will be popped from the
	// queues
	pq.RWMutex.Lock()
	pq.factor = factor
	pq.RWMutex.Unlock()
}

// Push a message on the queue, checking if it is a priority message type, and
// putting it into the channel relevant to the priority.
//
// If the channel buffer is full, this function will block, and it is recommended
// to increase the size specified in the constructor. If the buffer usage is too
// high or peaky, then there may be bottlenecks in the processing phase.
//
// The queue will auto-scale rather than fail by pausing operations and
// increasing the buffer sizes by 25% if autoGrow is set in the NewPriorityQueue
func (pq *PriorityQueue) Push(msg []byte) {
	pq.RWMutex.RLock()
	defer pq.RWMutex.RUnlock()
	if len(msg) > 0 {
		for _, v := range Priority {
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
// The rules for selection from the priority vs nonpriority queue are:
//
// - if only one queue has items, it is picked
// - if the number of items in priority queue is factor of factor, but the non-priority
//   also not at the same time, non-priority is selected, if the non-priority queue
//   has factor*X items then it is selected instead.
//
// In this way, both queues will empty favoring towards the more full queue, and
// ensuring both queues get emptied even if the one is filling up faster than the
// consumers are using them for a time.
func (pq *PriorityQueue) Pop() (msg []byte) {
	pq.RWMutex.RLock()
	defer pq.RWMutex.RUnlock()
	P, N := pq.Plen.Load(), pq.Nlen.Load()
	switch {
	case P == 0 && N == 0:
		return
	case P > 0 && N < P || (P%pq.factor == 0 && N%pq.factor != 0):
		msg = <-pq.Priority
		pq.Plen.Dec()
	case N > 0:
		msg = <-pq.Nonpriority
		pq.Nlen.Dec()
	}
	return
}
