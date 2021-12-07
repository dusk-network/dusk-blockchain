package peer

import (
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

const zero uint32 = 0
const qSize uint32 = 20

// freeList is a simple memory allocator to avoid GC during tests on raw queue processing
type freeList struct {
	buf   [][]byte
	alloc map[*[]byte]bool
}

func newFreeList(size int) (fl freeList) {
	fl.alloc = make(map[*[]byte]bool)
	fl.buf = make([][]byte, size)
	for i := range fl.buf {
		fl.buf[i] = make([]byte, 64)
	}
	for i := range fl.buf {
		fl.alloc[&fl.buf[i]] = false
	}
	return
}

func (fl *freeList) Alloc() *[]byte {
	for i := range fl.alloc {
		if !fl.alloc[i] {
			fl.alloc[i] = true
			return i
		}
	}
	panic("free list was not made big enough for purpose")
}

func (fl *freeList) Free(buf *[]byte) {
	fl.alloc[buf] = false
}

var freelist = newFreeList(100)

func genMessage(pri bool) (msg *[]byte) {
	//msg = make([]byte, 1000)
	msg = freelist.Alloc()
	if pri {
		// randomly select the value in one of the priority topics
		(*msg)[0] = byte(Priority[rand.Intn(len(Priority))])
	} else {
		(*msg)[0] = byte(topics.Version)
	}
	return
}

// TestPriorityQueue runs a set of basic tests to make sure the queue fills and
// empties correctly. Fills all queues equally, which means they empty equally,
// non-priority first
func TestPriorityQueue(t *testing.T) {
	q := NewPriorityQueue(int(qSize), 3)
	assert.Equal(t, zero, q.Plen.Load())
	assert.Equal(t, zero, q.Nlen.Load())
	for i := uint32(0); i < qSize; i++ {
		q.Push(*genMessage(true))
		q.Push(*genMessage(false))
	}
	assert.Equal(t, qSize, q.Plen.Load())
	assert.Equal(t, qSize, q.Nlen.Load())
	var counter int
	expected := `expected := []bool{false,true,false,true,false,true,false,true,false,true,false,true,false,true,false,true,false,true,false,true,false,true,false,true,false,true,false,true,false,true,false,true,false,true,false,true,false,true,false,true,}`
	out := "expected := []bool{"
	for msg := q.Pop(); msg != nil; msg = q.Pop() {
		if msg[0] == 0 {
			out += "false,"
		} else {
			out += "true,"
		}
		counter++
		freelist.Free(&msg)
	}
	out += "}"
	assert.Equal(t, out, expected)
	t.Logf("\n%s\n", out) // this will print what is coming out to put in the expected above
}

// TestPriorityQueuePriorityFlood tests queue behavior under conditions of DoS of
// priority messages. The goal of the test is to see how quickly the non-priority
// pool drains while 2x as many priority messages come in.
//
// In order to eliminate the play of the Go garbage collector, for this and the
// following test the foregoing freelist is used so no de-allocation is required
// for the lifetime of the test. A benchmark can be derived also from this code
// to determine the overhead incurred by the queue.
//
// This test uses factor=3 because it exactly copes with this 2x condition.
func TestPriorityQueuePriorityFlood(t *testing.T) {
	q := NewPriorityQueue(int(qSize), 3)
	assert.Equal(t, zero, q.Plen.Load())
	assert.Equal(t, zero, q.Nlen.Load())
	for i := uint32(0); i < qSize/2; i++ {
		q.Push(*genMessage(true))
		q.Push(*genMessage(true))
		q.Push(*genMessage(false))
	}
	assert.Equal(t, qSize, q.Plen.Load())
	assert.Equal(t, qSize/2, q.Nlen.Load())

}

// TestPriorityQueueNonPriorityFlood tests queue behavior under conditions of DoS
// of non-priority messages. This is the reverse of the other test. This is to
// demonstrate that the queue resists flooding via attacks on the priority scheme
// causing stale message queues.
func TestPriorityQueueNonPriorityFlood(t *testing.T) {

}
