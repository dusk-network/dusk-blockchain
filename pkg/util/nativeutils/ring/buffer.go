package ring

import (
	"runtime"
	"sync/atomic"
	"unsafe"
)

// Buffer represents a circular array of pointers.
type Buffer struct {
	items      []unsafe.Pointer
	writeIndex uint32
}

// Consumer represents an entity which can read items from a ring buffer.
// It maintains its own read index, and cache.
type Consumer struct {
	ring      *Buffer
	cache     *Cache
	readIndex uint32
}

// NewBuffer returns an initialized ring buffer.
func NewBuffer(length int) *Buffer {
	return &Buffer{
		items: make([]unsafe.Pointer, length),
	}
}

// Put an item on the ring buffer.
func (r *Buffer) Put(item []byte) {
	atomic.StorePointer(&r.items[atomic.LoadUint32(&r.writeIndex)], unsafe.Pointer(&item))
	incrementIndex(&r.writeIndex, len(r.items)-1)
}

// increment index. if index is at the end of the slice, set to 0
func incrementIndex(index *uint32, length int) {
	if !atomic.CompareAndSwapUint32(index, uint32(length), 0) {
		atomic.AddUint32(index, 1)
	}
}

// NewConsumer returns a Consumer, which can read from the passed Buffer.
func NewConsumer(ring *Buffer) *Consumer {
	return &Consumer{ring, NewCache(len(ring.items)), 0}
}

// Consume from an item from the ringbuffer and notifies whether it is already present in the cache
func (c *Consumer) Consume() ([]byte, bool) {
	found := true
	if b := c.getItem(); b != nil {
		if !c.cache.Has(b, c.readIndex) {
			c.cache.Put(b, c.readIndex)
			found = false
		}

		runtime.Gosched()
		incrementIndex(&c.readIndex, len(c.ring.items)-1)
		return b, found
	}

	return nil, true
}

func (c *Consumer) getItem() []byte {
	itemPtr := atomic.LoadPointer(&c.ring.items[atomic.LoadUint32(&c.readIndex)])
	if itemPtr != nil {
		return *(*[]byte)(itemPtr)
	}
	return nil
}
