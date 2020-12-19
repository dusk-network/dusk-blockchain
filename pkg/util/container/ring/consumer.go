// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package ring

import (
	"io"
)

// Consumer represents an entity which can read items from a ring buffer.
// It maintains its own read index, and cache.
type Consumer struct {
	ring *Buffer
	w    io.WriteCloser
	// consumes the retrieved data and returns true if no error
	// Returns false to terminate the consumer
	consume func(items [][]byte, w io.WriteCloser) bool
}

// NewConsumer returns a Consumer, which can read from the passed Buffer.
func NewConsumer(ring *Buffer, callback func(items [][]byte, w io.WriteCloser) bool, w io.WriteCloser) *Consumer {
	c := &Consumer{ring, w, callback}
	go c.run()

	return c
}

func (c *Consumer) run() {

	defer c.close()
	for {
		items, closed := c.ring.GetAll()
		if len(items) > 0 {
			if !c.consume(items, c.w) {
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
