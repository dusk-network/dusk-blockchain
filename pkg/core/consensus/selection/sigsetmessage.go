package selection

import (
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type sigSetMessage struct {
	WinningBlockHash []byte
	SigSet           []*msg.Vote
	SignedSigSet     []byte
	PubKeyBLS        []byte
	Round            uint64
	Step             uint8
}

// decodeSigSetMessage will decode a sigSetMessage struct from r,
// and return it.
func decodeSigSetMessage(r io.Reader) (*sigSetMessage, error) {
	var winningBlockHash []byte
	if err := encoding.Read256(r, &winningBlockHash); err != nil {
		return nil, err
	}

	voteSet, err := msg.DecodeVoteSet(r)
	if err != nil {
		return nil, err
	}

	var signedSigSet []byte
	if err := encoding.ReadBLS(r, &signedSigSet); err != nil {
		return nil, err
	}

	var pubKeyBLS []byte
	if err := encoding.ReadVarBytes(r, &pubKeyBLS); err != nil {
		return nil, err
	}

	var round uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &round); err != nil {
		return nil, err
	}

	var step uint8
	if err := encoding.ReadUint8(r, &step); err != nil {
		return nil, err
	}

	return &sigSetMessage{
		WinningBlockHash: winningBlockHash,
		SigSet:           voteSet,
		SignedSigSet:     signedSigSet,
		PubKeyBLS:        pubKeyBLS,
		Round:            round,
		Step:             step,
	}, nil
}
