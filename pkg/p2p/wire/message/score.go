package message

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

var _ wire.Event = (*Score)(nil)

type (
	// Score represents the Score Message with the fields consistent with the Blind Bid data structure
	Score struct {
		Score         []byte
		Proof         []byte
		Z             []byte
		BidListSubset []byte
		PrevHash      []byte
		Seed          []byte
		VoteHash      []byte
	}
)

// Equal as specified in the Event interface
func (e Score) Equal(ev wire.Event) bool {
	other, ok := ev.(Score)
	return ok && bytes.Equal(other.VoteHash, e.VoteHash)
}

// Sender of a Score event is the anonymous Z
func (e Score) Sender() []byte {
	return e.Z
}

// UnmarshalScore unmarshals the buffer into a Score Event
// Field order is the following:
// * Score Payload [score, proof, Z, BidList, Seed, Block Candidate Hash]
func UnmarshalScore(r *bytes.Buffer, ev wire.Event) error {
	sev := ev.(*Score)
	sev.Score = make([]byte, 32)
	if err := encoding.Read256(r, sev.Score); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &sev.Proof); err != nil {
		return err
	}

	sev.Z = make([]byte, 32)
	if err := encoding.Read256(r, sev.Z); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &sev.BidListSubset); err != nil {
		return err
	}

	sev.PrevHash = make([]byte, 32)
	if err := encoding.Read256(r, sev.PrevHash); err != nil {
		return err
	}

	sev.Seed = make([]byte, 33)
	if err := encoding.ReadBLS(r, sev.Seed); err != nil {
		return err
	}

	sev.VoteHash = make([]byte, 32)
	if err := encoding.Read256(r, sev.VoteHash); err != nil {
		return err
	}

	return nil
}

// MarshalScore the buffer into a committee Event
// Field order is the following:
// * Blind Bid Fields [Score, Proof, Z, BidList, Seed, Candidate Block Hash]
func MarshalScore(r *bytes.Buffer, ev wire.Event) error {
	sev, ok := ev.(*Score)
	if !ok {
		// sev is nil
		return nil
	}

	// Score
	if err := encoding.Write256(r, sev.Score); err != nil {
		return err
	}

	// Proof
	if err := encoding.WriteVarBytes(r, sev.Proof); err != nil {
		return err
	}

	// Z
	if err := encoding.Write256(r, sev.Z); err != nil {
		return err
	}

	// BidList
	if err := encoding.WriteVarBytes(r, sev.BidListSubset); err != nil {
		return err
	}

	if err := encoding.Write256(r, sev.PrevHash); err != nil {
		return err
	}

	// Seed
	if err := encoding.WriteBLS(r, sev.Seed); err != nil {
		return err
	}

	// CandidateHash
	if err := encoding.Write256(r, sev.VoteHash); err != nil {
		return err
	}
	return nil
}
