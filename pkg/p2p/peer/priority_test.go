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
		msg[0]=byte(Priority[rand.Intn(len(Priority))])
	} else {
		// just use the very first topic since anyway that makes it land in non-priority,
		// and because that is zero, nothing to be done
		//msg[0]=byte(topics.Version) // redundant since iota means the first value, ie zero.
	}
	return
}

// TestPriorityQueue runs a set of basic tests to make sure the queue fills and
// empties correctly
func TestPriorityQueue(t *testing.T) {
	q := NewPriorityQueue(20, 3)
	var zero uint32 = 0
	assert.Equal(t, zero, q.Plen.Load())
	assert.Equal(t, zero, q.Nlen.Load())
	t.Logf("testing testing\n")
}

// TestPriorityQueuePriorityFlood tests queue behavior under conditions of DoS of
// priority messages
func TestPriorityQueuePriorityFlood(t *testing.T) {

}

// TestPriorityQueueNonPriorityFlood tests queue behavior under conditions of DoS
// of non-priority messages
func TestPriorityQueueNonPriorityFlood(t *testing.T) {

}
