package score

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	// Event represents the Score Message with the fields consistent with the Blind Bid data structure
	Event struct {
		// Fields related to the consensus
		Round uint64
		Step  uint8

		// Fields related to the score
		Score         []byte
		Proof         []byte
		Z             []byte
		BidListSubset []byte
		Seed          []byte
		CandidateHash []byte
	}

	// UnMarshaller unmarshals consensus events. It is a helper to be embedded in the various consensus message unmarshallers
	UnMarshaller struct {
		validate func(*bytes.Buffer) error
	}
)

// Equal as specified in the Event interface
func (e *Event) Equal(ev wire.Event) bool {
	other, ok := ev.(*Event)
	return ok && other.Round == e.Round && bytes.Equal(other.CandidateHash, e.CandidateHash)
}

// Sender of a Score event is the anonymous Z
func (e *Event) Sender() []byte {
	return e.Z
}

// NewUnMarshaller creates a new Event UnMarshaller which takes care of Decoding and Encoding operations
func NewUnMarshaller(validate func(*bytes.Buffer) error) *UnMarshaller {
	return &UnMarshaller{validate}
}

// Unmarshal unmarshals the buffer into a Score Event
// Field order is the following:
// * Consensus Header [BLS Public Key; Round; Step]
// * Score Payload [score, proof, Z, BidList, Seed, Block Candidate Hash]
func (um *UnMarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	if err := um.validate(r); err != nil {
		return err
	}

	sev := ev.(*Event)

	// Decoding Round
	if err := encoding.ReadUint64(r, binary.LittleEndian, &sev.Round); err != nil {
		return err
	}

	// Decoding Step
	if err := encoding.ReadUint8(r, &sev.Step); err != nil {
		return err
	}

	if err := encoding.Read256(r, &sev.Score); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &sev.Proof); err != nil {
		return err
	}

	if err := encoding.Read256(r, &sev.Z); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &sev.BidListSubset); err != nil {
		return err
	}

	if err := encoding.ReadBLS(r, &sev.Seed); err != nil {
		return err
	}

	if err := encoding.Read256(r, &sev.CandidateHash); err != nil {
		return err
	}

	return nil
}

// Marshal the buffer into a committee Event
// Field order is the following:
// * Consensus Header [Round; Step]
// * Blind Bid Fields [Score, Proof, Z, BidList, Seed, Candidate Block Hash]
func (um *UnMarshaller) Marshal(r *bytes.Buffer, ev wire.Event) error {
	sev := ev.(*Event)

	if err := encoding.WriteUint64(r, binary.LittleEndian, sev.Round); err != nil {
		return err
	}

	if err := encoding.WriteUint8(r, sev.Step); err != nil {
		return err
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

	if err := encoding.WriteVarBytes(r, sev.BidListSubset); err != nil {
		return err
	}

	if err := encoding.WriteBLS(r, sev.Seed); err != nil {
		return err
	}

	// CandidateHash
	if err := encoding.Write256(r, sev.CandidateHash); err != nil {
		return err
	}
	return nil
}
