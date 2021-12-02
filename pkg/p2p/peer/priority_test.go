package peer

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

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
