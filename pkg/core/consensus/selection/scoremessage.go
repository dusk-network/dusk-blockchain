package selection

import (
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type scoreMessage struct {
	Score         []byte
	Proof         []byte
	Z             []byte
	BidListSubset []byte
	Seed          []byte
	CandidateHash []byte
	Round         uint64
	Step          uint8
}

// decodeScoreMessage will decode a scoreMessage struct from r,
// and return it.
func decodeScoreMessage(r io.Reader) (*scoreMessage, error) {
	var score []byte
	if err := encoding.Read256(r, &score); err != nil {
		return nil, err
	}

	var proof []byte
	if err := encoding.ReadVarBytes(r, &proof); err != nil {
		return nil, err
	}

	var z []byte
	if err := encoding.Read256(r, &z); err != nil {
		return nil, err
	}

	var BidListSubset []byte
	if err := encoding.ReadVarBytes(r, &BidListSubset); err != nil {
		return nil, err
	}

	var seed []byte
	if err := encoding.ReadBLS(r, &seed); err != nil {
		return nil, err
	}

	var candidateHash []byte
	if err := encoding.Read256(r, &candidateHash); err != nil {
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

	return &scoreMessage{
		Score:         score,
		Proof:         proof,
		Z:             z,
		BidListSubset: BidListSubset,
		Seed:          seed,
		CandidateHash: candidateHash,
		Round:         round,
		Step:          step,
	}, nil
}
