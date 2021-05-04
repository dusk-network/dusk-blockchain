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

type Addr struct {
	NetAddr string
}

func (a Addr) Copy() payload.Safe {
	return Addr{a.NetAddr}
}

func UnmarshalAddrMessage(r *bytes.Buffer, m SerializableMessage) {
	a := Addr{r.String()}
	m.SetPayload(a)
}
