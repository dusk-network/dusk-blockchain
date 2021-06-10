// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package ring

import (
	"sort"
)

type Writer interface {
	Write(e Elem) (int, error)
	Close() error
}

// Consumer represents an entity which can read items from a ring buffer.
// It maintains its own read index, and cache.
type Consumer struct {
	ring *Buffer
	w    Writer
	// consumes the retrieved data and returns true if no error
	// Returns false to terminate the consumer
	consume func(items []Elem, w Writer) bool

	sortByPriority bool
}

// NewConsumer returns a Consumer, which can read from the passed Buffer.
func NewConsumer(ring *Buffer, callback func(items []Elem, w Writer) bool, w Writer, sortByPriority bool) *Consumer {
	c := &Consumer{
		ring:           ring,
		w:              w,
		consume:        callback,
		sortByPriority: sortByPriority,
	}

	go c.run()

	return c
}

func (c *Consumer) run() {
	defer c.close()

	for {
		elems, closed := c.ring.GetAll()
		if len(elems) > 0 {
			if c.sortByPriority {
				sort.Sort(elems)
			}

			if !c.consume(elems, c.w) {
				return
			}
		}

		if closed {
			return
		}
	}
}

func (c *Consumer) close() {
	if c.ring != nil {
		c.ring.Close()
	}

	if c.w != nil {
		_ = c.w.Close()
	}
}
