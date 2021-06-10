// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package ring

import (
	"bytes"
	"sync"
	"sync/atomic"
)

// Elem single data unit of a ring buffer.
type Elem struct {
	Data     []byte
	Header   []byte
	Priority byte
}

// Buffer represents a circular array of data items.
// It is suitable for (single/multiple) consumers (single/multiple) producers data transfer.
type Buffer struct {
	items    []Elem
	mu       *sync.Mutex
	notEmpty *sync.Cond

	// writeIndex
	wri    int32
	closed syncBool
}

// NewBuffer returns an initialized ring buffer.
func NewBuffer(length int) *Buffer {
	m := &sync.Mutex{}
	cv := sync.NewCond(m)

	return &Buffer{
		items:    make([]Elem, length),
		notEmpty: cv,
		mu:       m,
		wri:      -1,
	}
}

// Put an item on the ring buffer.
func (r *Buffer) Put(e Elem) bool {
	if e.Data == nil || r.closed.Load() {
		return false
	}

	// Protect the slice and the writeIndex
	r.mu.Lock()

	if !r.Has(e) {
		// Store the new item
		r.wri++
		// Reset the writeIndex as this is ringBuffer
		if int(r.wri) == len(r.items) {
			r.wri = 0
		}

		r.items[r.wri] = e
	}

	r.mu.Unlock()

	// Signal consumers we've got new item
	r.notEmpty.Signal()

	return true
}

// Close will close the Buffer.
func (r *Buffer) Close() {
	r.closed.Store(true)
	// Signal all consumers for the state change
	r.notEmpty.Broadcast()
}

// GetAll gets all items in a buffer.
func (r *Buffer) GetAll() (ElemArray, bool) {
	r.mu.Lock()

	for int(r.wri) < 0 && !r.closed.Load() {
		r.notEmpty.Wait()
	}

	elements := make(ElemArray, 0, len(r.items))

	for i, elem := range r.items {
		if elem.Data == nil {
			break
		}

		elements = append(elements, elem)
		r.items[i].Data = nil
	}

	r.wri = -1
	r.mu.Unlock()

	return elements, r.closed.Load()
}

// Has checks if item exists.
func (r *Buffer) Has(e Elem) bool {
	if e.Data == nil {
		return false
	}

	for _, bufElem := range r.items {
		if bytes.Equal(bufElem.Data, e.Data) {
			return true
		}
	}

	return false
}

// Closed checks if buffer is closed.
func (r *Buffer) Closed() bool {
	return r.closed.Load()
}

// ElemArray a sortable array of Elem.
type ElemArray []Elem

// Len complies with the Sort interface.
func (e ElemArray) Len() int { return len(e) }

// Swap complies with the Sort interface.
func (e ElemArray) Swap(i, j int) { e[i], e[j] = e[j], e[i] }

// Less complies with the Sort interface.
func (e ElemArray) Less(i, j int) bool { return e[i].Priority < e[j].Priority }

// syncBool provides atomic Load/Store for bool type.
type syncBool struct {
	value int32
}

func (s *syncBool) Store(value bool) {
	i := int32(0)
	if value {
		i = 1
	}

	atomic.StoreInt32(&(s.value), i)
}

func (s *syncBool) Load() bool {
	return atomic.LoadInt32(&(s.value)) != 0
}
