package peer

import (
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

func genMessage(pri bool) (msg []byte) {
	msg = make([]byte, 1000)
	if pri {
		// randomly select the value in one of the priority topics
		msg[0] = byte(Priority[rand.Intn(len(Priority))])
	} else {
		// just use the very first topic since anyway that makes it land in non-priority,
		// and because that is zero, nothing to be done
		//msg[0]=byte(topics.Version) // redundant since iota means the first value, ie zero.
	}
	return
}

// TestPriorityQueue runs a set of basic tests to make sure the queue fills and
// empties correctly. Fills all queues equally, which means they empty equally,
// non-priority first
func TestPriorityQueue(t *testing.T) {
	var zero uint32 = 0
	var qSize uint32 = 20
	q := NewPriorityQueue(int(qSize), 3)
	assert.Equal(t, zero, q.Plen.Load())
	assert.Equal(t, zero, q.Nlen.Load())
	for i := uint32(0); i < qSize; i++ {
		q.Push(genMessage(true))
		q.Push(genMessage(false))
	}
	assert.Equal(t, qSize, q.Plen.Load())
	assert.Equal(t, qSize, q.Nlen.Load())
	var counter int
	for msg := q.Pop(); msg != nil; msg = q.Pop() {
		assert.Equal(t, msg[0]==0, counter%2==0)
		//if msg[0] == 0 {
		//	t.Logf("popping non-priority")
		//} else {
		//	t.Logf("popping priority")
		//}
		counter++
	}
}

// TestPriorityQueuePriorityFlood tests queue behavior under conditions of DoS of
// priority messages. The goal of the test is to see how quickly the nonpriority
// pool drains while 2x as many priority messages come in.
//
// In order to eliminate the play of the Go garbage collector, for this and the
// following test the foregoing freelist is used so no deallocation is required
// for the lifetime of the test. A benchmark can be derived also from this code
// to determine the overhead incurred by the queue.
func TestPriorityQueuePriorityFlood(t *testing.T) {

}

// TestPriorityQueueNonPriorityFlood tests queue behavior under conditions of DoS
// of non-priority messages. This is the reverse of the other test. This is to
// demonstrate that the queue resists flooding via attacks on the priority scheme
// causing stale message queues.
func TestPriorityQueueNonPriorityFlood(t *testing.T) {

}
