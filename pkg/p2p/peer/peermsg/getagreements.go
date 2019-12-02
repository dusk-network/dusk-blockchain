package peermsg

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

type GetAgreements struct {
	Round uint64
}

func (g *GetAgreements) Encode(w *bytes.Buffer) error {
	return encoding.WriteUint64LE(w, g.Round)
}

func (g *GetAgreements) Decode(r *bytes.Buffer) error {
	return encoding.ReadUint64LE(r, &g.Round)
}
