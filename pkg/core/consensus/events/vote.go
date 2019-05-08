package events

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type Vote struct {
	Round     uint64
	Step      uint8
	BlockHash []byte
}

// MarshalSignableVote marshals a Vote
func MarshalSignableVote(r *bytes.Buffer, vote *Vote) error {
	if err := encoding.WriteUint64(r, binary.LittleEndian, vote.Round); err != nil {
		return err
	}

	if err := encoding.WriteUint8(r, vote.Step); err != nil {
		return err
	}

	if err := encoding.Write256(r, vote.BlockHash); err != nil {
		return err
	}
	return nil
}
