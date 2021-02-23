// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package responding

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

// ProcessPing will simply return a Pong message.
// Satisfies the peer.ProcessorFunc interface.
func ProcessPing(srcPeerID string, _ message.Message) ([]bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	if err := topics.Prepend(buf, topics.Pong); err != nil {
		return nil, err
	}

	return []bytes.Buffer{*buf}, nil
}

// ProcessPong empty processor to process a Pong message
// Satisfies the peer.ProcessorFunc interface.
func ProcessPong(srcPeerID string, _ message.Message) ([]bytes.Buffer, error) {
	return []bytes.Buffer{}, nil
}
