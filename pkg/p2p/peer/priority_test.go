package peer

import (
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

const zero uint32 = 0
const qSize uint32 = 40

func genMessage(pri bool) (msg *[]byte) {
	msgb := make([]byte, 64)
	//msg = freelist.Alloc()
	msg = &msgb
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
	rand.Seed(0xdeadbeeffacade)
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
	}
	out += "}"
	assert.Equal(t, out, expected)
	t.Logf("\n%s\n", out) // this will print what is coming out to put in the expected above
}

// TestPriorityQueuePriorityFlood tests queue behavior under conditions of DoS of
// priority messages. The goal of the test is to see how quickly the non-priority
// pool drains while 2x as many priority messages come in.
func TestPriorityQueuePriorityFlood(t *testing.T) {
	rand.Seed(0xdeadbeeffacade)
	q := NewPriorityQueue(int(qSize), 4)
	for i := uint32(0); i < qSize/3; i++ {
		q.Push(*genMessage(true))
		q.Push(*genMessage(true))
		q.Push(*genMessage(false))
	}
	expected := "expected := []bool{true, true, true, false, true, true, false, true, true, false, true, true, false, true, true, true, true, true, true, false, true, true, false, true, true, false, true, true, false, true, true, true, true, true, true, false, true, true, false, true, true, false, true, true, false, true, true, true, true, false, true, false, true, true, false, true, true, false, true, false, false, true, true, true, true, false, true, false, true, true, false, true, true, false, true, false, false, true, true, true, true, false, true, false, true, true, false, true, true, false, true, false, false, true, true, true, true, false, true, false, true, true, false, true, true, false, true, false, false, true, true, true, true, false, true, false, true, true, false, true, true, false, true, false, false, true, true, true, true, false, true, false, true, true, false, true, true, false, true, false, false, }"
	//t.Logf("%d, %d\n", q.Plen.Load(), q.Nlen.Load())
	out := "expected := []bool{"
	// Now, run a loop that pops four and pushes three
	var msg []byte
oot:
	for i := 0; i < 100; i++ {
		for j := 0; j < 4; j++ {
			msg = q.Pop()
			if msg == nil {
				break oot
			}
			//t.Logf("%d, %d\n", q.Plen.Load(), q.Nlen.Load())
			if IsPriority(msg[0]) {
				out += "true, "
			} else {
				out += "false, "
			}
		}
		q.Push(*genMessage(true))
		//t.Logf("%d, %d\n", q.Plen.Load(), q.Nlen.Load())
		q.Push(*genMessage(true))
		//t.Logf("%d, %d\n", q.Plen.Load(), q.Nlen.Load())
		q.Push(*genMessage(false))
		//t.Logf("%d, %d\n", q.Plen.Load(), q.Nlen.Load())
	}
	out += "}"
	assert.Equal(t, expected, out)
	//t.Logf("\n%s\n", out) // this will print what is coming out to put in the expected above
}

// TestPriorityQueueNonPriorityFlood tests queue behavior under conditions of DoS
// of non-priority messages. This is the reverse of the other test. This is to
// demonstrate that the queue resists flooding via attacks on the priority scheme
// causing stale message queues.
func TestPriorityQueueNonPriorityFlood(t *testing.T) {
	rand.Seed(0xdeadbeeffacade)

}
