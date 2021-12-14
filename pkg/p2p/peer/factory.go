// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package peer

import "context"

// ReaderFactory is responsible for spawning peers. It provides them with the
// reference to the message processor, which will process the received messages.
type ReaderFactory struct {
	processor *MessageProcessor
}

// NewReaderFactory returns an initialized ReaderFactory.
func NewReaderFactory(processor *MessageProcessor) *ReaderFactory {
	return &ReaderFactory{processor}
}

// SpawnReader returns a Reader. It will still need to be launched by
// running ReadLoop in a goroutine.
func (f *ReaderFactory) SpawnReader(conn *Connection, ctx context.Context) *Reader {
	reader := &Reader{
		Connection: conn,
		processor:  f.processor,
		priority:   NewPriorityQueue(ctx, 32, 4),
	}

	return reader
}
