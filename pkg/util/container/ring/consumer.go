package ring

import "io"

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
	go func() {
		defer func() {
			c.close()
		}()

		for {
			items, close := c.ring.GetAll()
			if len(items) > 0 {
				if !c.consume(items, c.w) {
					return
				}
			}

			if close {
				return
			}
		}
	}()

	return c
}

func (c *Consumer) close() {
	c.ring.Close()
	if c.w != nil {
		_ = c.w.Close()
	}
}
