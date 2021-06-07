// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package ring

import (
	"bytes"
	"sync"
	"testing"
	"time"
)

func TestRingBuffer100(t *testing.T) {
	testConsumerProducer(t, 100, 1, 50)
}

func TestRingBuffer1000(t *testing.T) {
	testConsumerProducer(t, 1000, 1, 50)
}

func TestSingleConsumerSingleProducer(t *testing.T) {
	testConsumerProducer(t, 1000, 1, 1)
}

func TestMultipleConsumersSingleProducer(t *testing.T) {
	testConsumerProducer(t, 1000, 10, 1)
}

func TestMultipleConsumersMultipleProducers(t *testing.T) {
	testConsumerProducer(t, 1000, 10, 100)
}

// Safe array of arrays.
type safeSlice struct {
	data []Entry
	mu   sync.RWMutex
}

func (s *safeSlice) append(items []Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data == nil {
		s.data = make([]Entry, 0)
	}

	s.data = append(s.data, items...)
}

func (s *safeSlice) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

func (s *safeSlice) Equal(b *safeSlice) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, sItem := range s.data {
		var found bool

		b.mu.RLock()
		for _, bItem := range b.data {
			if bytes.Equal(sItem.Data, bItem.Data) {
				found = true
				break
			}
		}

		b.mu.RUnlock()

		if !found {
			return false
		}
	}

	return true
}

// testConsumerProducer instantiates a ring buffer, Consumer(s) and Producers.
func testConsumerProducer(t *testing.T, bufferSize int, consumersNum int, producersNum int) {
	ring := NewBuffer(bufferSize)

	// items slice read from ring buffer
	var rItems safeSlice

	consumeFunc := func(items []Entry, w Writer) bool {
		rItems.append(items)
		return true
	}

	// Init 1 or many consumers
	for i := 0; i < consumersNum; i++ {
		_ = NewConsumer(ring, consumeFunc, nil)
	}

	time.Sleep(500 * time.Millisecond)

	// items written to the ring buffer
	var wItems safeSlice

	// Init a producer
	producer := func(id int, wg *sync.WaitGroup) {
		data := make([]Entry, 20*consumersNum)
		for j := 0; j < len(data); j++ {
			data[j].Data = []byte{byte(id + j)}
			// put some bytes on the ring
			ring.Put(data[j])
		}

		wItems.append(data)
		wg.Done()
	}

	var wg sync.WaitGroup

	wg.Add(producersNum)

	// Multiple producers putting data
	for i := 1; i <= producersNum; i++ {
		go producer(i+100, &wg)
	}

	// wait all items to be written
	wg.Wait()

	// Ensure all produced items have been consumed

	if wItems.Len() == 0 {
		t.Fatalf("count of items written to the ring buffer shoud not be 0")
	}

	// All written items should read (as result stored in rItems)
	retry := 0

	for {
		retry++
		if retry == 4 {
			t.Fatal("not all written items have been read")
		}

		if wItems.Len() > 0 && wItems.Equal(&rItems) {
			break
		}

		time.Sleep(1 * time.Second)
	}

	// Ask the consumer to terminate
	ring.Close()

	if ring.Closed() != true {
		t.Error("not closed ring")
	}

	// give consumers time to terminate
	time.Sleep(10 * time.Millisecond)
}
