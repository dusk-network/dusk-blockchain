package ring

import (
	"bytes"
	"io"
	"sync"
	"testing"
	"time"
)

func TestRingBuffer100(t *testing.T) {
	mainTest(t, 100)
}

func TestRingBuffer1000(t *testing.T) {
	mainTest(t, 1000)
}

// 50 producers sending 20 items concurrently
func mainTest(t *testing.T, bufferSize int) {
	var consumeMu sync.RWMutex
	consumedItems := make([][]byte, 0)
	consumeFunc := func(items [][]byte, w io.WriteCloser) bool {

		consumeMu.Lock()
		consumedItems = append(consumedItems, items...)
		consumeMu.Unlock()

		return true
	}

	ring := NewBuffer(bufferSize)

	// Init consumers
	_ = NewConsumer(ring, consumeFunc, nil)

	time.Sleep(500 * time.Millisecond)

	// Init producers
	var producedMu sync.RWMutex
	producedItems := make([][]byte, 0)

	producer := func(id int) {
		data := make([][]byte, 20)

		for j := 0; j < len(data); j++ {
			data[j] = []byte{byte(id + j)}
			// put some bytes on the ring
			ring.Put(data[j])
		}

		producedMu.Lock()
		producedItems = append(producedItems, data...)
		producedMu.Unlock()
	}

	// Multiple producers putting data
	for i := 1; i <= 50; i++ {
		go producer(i + 100)
	}

	// wait all to be consumed
	time.Sleep(100 * time.Millisecond)

	// Ensure all produced items have been consumed
	consumeMu.RLock()
	producedMu.RLock()

	if len(producedItems) == 0 {
		t.Fatalf("expecting more produced item")
	}

	for i, producedItem := range producedItems {
		var found bool
		for _, consumedItem := range consumedItems {
			if bytes.Equal(producedItem, consumedItem) {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("expecting all produced items to be consumed within a time frame (ind %d, from %d)", i, len(consumedItems))
		}
	}
	producedMu.RUnlock()
	consumeMu.RUnlock()

	// Ask the consumer to terminate
	ring.Close()

	// give consumers time to terminate
	time.Sleep(10 * time.Millisecond)
}
