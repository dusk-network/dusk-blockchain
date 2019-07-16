package ring

import (
	"bytes"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var _signal struct{}

func pollData(b *Buffer, done chan struct{}) *bytes.Buffer {
	buf := new(bytes.Buffer)
	c := NewConsumer(b)
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				data, found := c.Consume()
				if !found {
					_, _ = buf.Write(data)
				}
				time.Sleep(2 * time.Millisecond)
			}
		}
	}()
	return buf
}

// Try to force a panic regarding index being out of range. If `Put` is not synchronized,
// this test should cause an `index out of range` error
func TestRingBufferRace(t *testing.T) {
	b := NewBuffer(10)

	for i := 0; i < 1000; i++ {
		go func() {
			for i := 0; i < 1000; i++ {
				b.Put([]byte{0, 1, 2, 3, 4})
			}
		}()
	}
}

func TestRingBuffer(t *testing.T) {
	consumers := 3
	bufs := make([]*bytes.Buffer, consumers)
	b := NewBuffer(10)
	done := make(chan struct{})

	// add consumers
	for i := 0; i < consumers; i++ {
		bufs[i] = pollData(b, done)
	}

	// put some bytes on the ring
	b.Put([]byte{1})
	b.Put([]byte{2})
	b.Put([]byte{3})
	b.Put([]byte{4})

	// wait a bit for the consumers to consume it
	time.Sleep(200 * time.Millisecond)

	for i := 0; i < consumers; i++ {
		done <- _signal
	}
	time.Sleep(200 * time.Millisecond)

	// buffers should all be equal and be [1, 2, 3, 4]
	assert.Equal(t, []byte{1, 2, 3, 4}, bufs[0].Bytes())
	for i := 0; i < consumers; i++ {
		if i == 0 {
			continue
		}

		assert.Equal(t, bufs[i-1], bufs[i])
	}
}

func BenchmarkPut(b *testing.B) {
	ring := NewBuffer(10)
	for i := 0; i < b.N; i++ {
		ring.Put([]byte{1, 1, 1, 1})
	}
}

func BenchmarkGet(b *testing.B) {
	ring := NewBuffer(b.N)
	for i := 0; i < b.N; i++ {
		ring.Put([]byte{1, 1, 1, 1})
	}

	c := NewConsumer(ring)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.getItem()
		incrementIndex(&c.readIndex, len(c.ring.items)-1)
	}
}

func BenchmarkGetPut(b *testing.B) {
	ring := NewBuffer(200)

	counter := uint32(0)
	wg := sync.WaitGroup{}
	wg.Add(1)

	c := NewConsumer(ring)
	go func() {
		defer wg.Done()
		for {
			_ = c.getItem()
			if atomic.AddUint32(&counter, 1) == uint32(b.N) {
				return
			}
		}
	}()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ring.Put([]byte{1, 1, 1, 1})
	}

	wg.Wait()
}
