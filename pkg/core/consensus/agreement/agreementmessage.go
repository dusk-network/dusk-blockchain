package agreement

import (
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// agreementMessage defines an agreement message on the Dusk wire protocol.
type agreementMessage struct {
	VoteSet       []*msg.Vote
	SignedVoteSet []byte
	PubKeyBLS     []byte
	BlockHash     []byte
	SigSetHash    []byte
	Round         uint64
	Step          uint8
}

// DecodeAgreementMessage will decode a scoreMessage struct from r,
// and return it.
func DecodeAgreementMessage(r io.Reader) (*agreementMessage, error) {
	voteSet, err := msg.DecodeVoteSet(r)
	if err != nil {
		return nil, err
	}

	var signedVoteSet []byte
	if err := encoding.ReadBLS(r, &signedVoteSet); err != nil {
		return nil, err
	}

	var pubKeyBLS []byte
	if err := encoding.ReadVarBytes(r, &pubKeyBLS); err != nil {
		return nil, err
	}

	var blockHash []byte
	if err := encoding.Read256(r, &blockHash); err != nil {
		return nil, err
	}

	var sigSetHash []byte
	if err := encoding.Read256(r, &sigSetHash); err != nil {
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

	return &agreementMessage{
		VoteSet:       voteSet,
		SignedVoteSet: signedVoteSet,
		PubKeyBLS:     pubKeyBLS,
		BlockHash:     blockHash,
		SigSetHash:    sigSetHash,
		Round:         round,
		Step:          step,
	}, nil
}
