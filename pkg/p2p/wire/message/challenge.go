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

// Challenge represents a challenge sent by a voucher seeder on connection.
type Challenge struct {
	ChallengeString []byte
}

// Copy a Challenge.
// Implements the payload.Safe interface.
func (c Challenge) Copy() payload.Safe {
	d := make([]byte, len(c.ChallengeString))
	copy(d, c.ChallengeString)
	return Challenge{d}
}

// UnmarshalChallengeMessage into a SerializableMessage.
func UnmarshalChallengeMessage(r *bytes.Buffer, m SerializableMessage) {
	c := Challenge{r.Bytes()}
	m.SetPayload(c)
}
