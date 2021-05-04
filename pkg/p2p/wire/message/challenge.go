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

type Challenge struct {
	ChallengeString []byte
}

func (c Challenge) Copy() payload.Safe {
	d := make([]byte, len(c.ChallengeString))
	copy(d, c.ChallengeString)
	return Challenge{d}
}

func UnmarshalChallengeMessage(r *bytes.Buffer, m SerializableMessage) {
	c := Challenge{r.Bytes()}
	m.SetPayload(c)
}
