package ring

import (
	"bytes"
	"sync"
)

// Buffer represents a circular array of data items.
type Buffer struct {
	items      [][]byte
	mu         *sync.Mutex
	notEmpty   *sync.Cond
	notFull    *sync.Cond
	writeIndex int32
	close      bool
}

// NewBuffer returns an initialized ring buffer.
func NewBuffer(length int) *Buffer {

	m := &sync.Mutex{}
	cv := sync.NewCond(m)
	cv2 := sync.NewCond(m)

	return &Buffer{
		items:      make([][]byte, length, length),
		notEmpty:   cv,
		notFull:    cv2,
		mu:         m,
		writeIndex: -1,
	}
}

// Put an item on the ring buffer.
func (r *Buffer) Put(item []byte) {

	if item == nil || r.close {
		return
	}

	// Protect the slice and the writeIndex
	r.mu.Lock()

	if !r.Has(item) {
		// Store the new item
		r.writeIndex++
		// Reset the writeIndex as this is ringBuffer
		if int(r.writeIndex) == len(r.items) {
			r.writeIndex = 0
		}

		r.items[r.writeIndex] = item
	}

	r.mu.Unlock()

	// Signal consumer we've got new item
	r.notEmpty.Signal()
}

func (r *Buffer) Close() {

	r.mu.Lock()
	r.close = true
	r.mu.Unlock()

	// Signal consumer for the state change
	r.notEmpty.Signal()
}

func (r *Buffer) GetAll() ([][]byte, bool) {

	r.mu.Lock()

	for int(r.writeIndex) < 0 && !r.close {
		r.notEmpty.Wait()
	}

	items := make([][]byte, 0)
	for i, itemPtr := range r.items {
		if itemPtr == nil {
			break
		}
		items = append(items, itemPtr)
		r.items[i] = nil
	}

	r.writeIndex = -1
	r.mu.Unlock()

	return items, r.close
}

func (r *Buffer) Has(item []byte) bool {

	if item == nil {
		return false
	}

	for _, existing := range r.items {
		if existing == nil {
			return false
		}
		if bytes.Equal(existing, item) {
			return true
		}
	}

	return false
}

func (r *Buffer) Closed() bool {
	r.mu.Lock()
	closed := r.close
	r.mu.Unlock()
	return closed
}
