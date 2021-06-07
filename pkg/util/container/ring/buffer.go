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

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

type MsgPriority byte

const (
	Low MsgPriority = iota
	High
)

type Entry struct {
	Data     []byte
	Header   []byte
	Topic    topics.Topic
	Priority MsgPriority
}

// Buffer represents a circular array of data items.
// It is suitable for (single/multiple) consumers (single/multiple) producers data transfer.
type Buffer struct {
	items      []Entry
	mu         *sync.Mutex
	notEmpty   *sync.Cond
	writeIndex int32
	closed     syncBool
}

// NewBuffer returns an initialized ring buffer.
func NewBuffer(length int) *Buffer {
	m := &sync.Mutex{}
	cv := sync.NewCond(m)

	return &Buffer{
		items:      make([]Entry, length),
		notEmpty:   cv,
		mu:         m,
		writeIndex: -1,
	}
}

// Put an item on the ring buffer.
func (r *Buffer) Put(i Entry) bool {
	if i.Data == nil || r.closed.Load() {
		return false
	}

	// Protect the slice and the writeIndex
	r.mu.Lock()

	if !r.Has(i.Data) {
		// Store the new item
		r.writeIndex++
		// Reset the writeIndex as this is ringBuffer
		if int(r.writeIndex) == len(r.items) {
			r.writeIndex = 0
		}

		r.items[r.writeIndex] = i

		// Simple sorting by priority. If most recent item is High Priority
		// then search for any Low Priority message pushed before. If found, swap
		// them.
		if r.items[r.writeIndex].Priority == High {
			for i := 0; i < len(r.items); i++ {
				if r.items[i].Data == nil {
					continue
				}

				if r.items[i].Priority == Low {
					// found Low Priority message
					// swap them
					r.items[i], r.items[r.writeIndex] = r.items[r.writeIndex], r.items[i]
					break
				}
			}
		}
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
func (r *Buffer) GetAll() ([]Entry, bool) {
	r.mu.Lock()

	for int(r.writeIndex) < 0 && !r.closed.Load() {
		r.notEmpty.Wait()
	}

	items := make([]Entry, 0)
	cap := 0

	for ind, item := range r.items {
		cap++
		if cap == 1000 {
			break
		}

		if item.Data == nil {
			break
		}

		items = append(items, item)
		r.items[ind].Data = nil
	}

	r.writeIndex = -1
	r.mu.Unlock()

	return items, r.closed.Load()
}

// Has checks if item exists.
func (r *Buffer) Has(item []byte) bool {
	if item == nil {
		return false
	}

	for _, bufItem := range r.items {
		if bytes.Equal(bufItem.Data, item) {
			return true
		}
	}

	return false
}

// Closed checks if buffer is closed.
func (r *Buffer) Closed() bool {
	return r.closed.Load()
}

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
