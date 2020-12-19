// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package message

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
)

// GetCandidate is used to request certain candidates from peers.
type GetCandidate struct {
	Hash []byte
}

// Copy a GetCandidate message.
// Implements the payload.Safe interface.
func (g GetCandidate) Copy() payload.Safe {
	h := make([]byte, len(g.Hash))
	copy(h, g.Hash)
	return GetCandidate{h}
}

// UnmarshalGetCandidateMessage into a SerializableMessage.
func UnmarshalGetCandidateMessage(r *bytes.Buffer, m SerializableMessage) {
	g := GetCandidate{r.Bytes()}
	m.SetPayload(g)
}
