package selection

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	// ScoreEvent represents the Score Message with the fields consistent with the Blind Bid data structure
	ScoreEvent struct {
		// Fields related to the consensus
		Round uint64
		Step  uint8

		// Fields related to the score
		Score         []byte
		Proof         []byte
		Z             []byte
		BidListSubset []byte
		Seed          []byte
		VoteHash      []byte
	}

	// ScoreUnMarshaller unmarshals consensus events. It is a helper to be embedded in the various consensus message unmarshallers
	ScoreUnMarshaller struct {
		validateFunc func(*bytes.Buffer) error
	}
)

// Equal as specified in the Event interface
func (e *ScoreEvent) Equal(ev wire.Event) bool {
	other, ok := ev.(*ScoreEvent)
	return ok && other.Round == e.Round && bytes.Equal(other.VoteHash, e.VoteHash)
}

// Sender of a Score event is the anonymous Z
func (e *ScoreEvent) Sender() []byte {
	return e.Z
}

// newScoreUnMarshaller creates a new Event UnMarshaller which takes care of Decoding and Encoding operations
func newScoreUnMarshaller() *ScoreUnMarshaller {
	return &ScoreUnMarshaller{}
}

// Unmarshal unmarshals the buffer into a Score Event
// Field order is the following:
// * Consensus Header [Round; Step]
// * Score Payload [score, proof, Z, BidList, Seed, Block Candidate Hash]
func (um *ScoreUnMarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	// check if the buffer has contents first
	// if not, we did not get any messages this round
	// TODO: review this
	if r.Len() == 0 {
		return nil
	}

	if err := um.validateFunc(r); err != nil {
		return err
	}

	sev := ev.(*ScoreEvent)

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

	if err := encoding.Read256(r, &sev.VoteHash); err != nil {
		return err
	}

	return nil
}

// Marshal the buffer into a committee Event
// Field order is the following:
// * Consensus Header [Round; Step]
// * Blind Bid Fields [Score, Proof, Z, BidList, Seed, Candidate Block Hash]
func (um *ScoreUnMarshaller) Marshal(r *bytes.Buffer, ev wire.Event) error {
	// TODO: review
	sev, ok := ev.(*ScoreEvent)
	if !ok {
		// ev is nil
		return nil
	}

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

	// BidList
	if err := encoding.WriteVarBytes(r, sev.BidListSubset); err != nil {
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
