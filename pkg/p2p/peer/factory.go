// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package peer

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

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
func (f *ReaderFactory) SpawnReader(conn *Connection, responseChan chan<- bytes.Buffer) *Reader {
	reader := &Reader{
		Connection:   conn,
		responseChan: responseChan,
		processor:    f.processor,
	}

	// On each new connection the node sends topics.Mempool to retrieve mempool
	// txs from the new peer
	go func() {
		responseChan <- topics.MemPool.ToBuffer()
	}()

	return reader
}
