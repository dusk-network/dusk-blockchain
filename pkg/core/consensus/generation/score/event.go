package score

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	zkproof "github.com/dusk-network/dusk-zkproof"
)

// TODO: consider embedding this into message.Score
type Event struct {
	header.Header

	Proof zkproof.ZkProof
	Seed  []byte
}

// New creates a new ScoreEvent for internal propagation only
func New() *Event {
	return &Event{
		Header: header.Header{},
	}
}

// TODO: get rid of serialization
func Marshal(b *bytes.Buffer, s Event) error {
	if err := encoding.WriteVarBytes(b, s.Proof.Proof); err != nil {
		return err
	}

	if err := encoding.Write256(b, s.Proof.Score); err != nil {
		return err
	}

	if err := encoding.Write256(b, s.Proof.Z); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(b, s.Proof.BinaryBidList); err != nil {
		return err
	}

	return encoding.WriteBLS(b, s.Seed)
}

func Unmarshal(b *bytes.Buffer, s *Event) error {
	if err := encoding.ReadVarBytes(b, &s.Proof.Proof); err != nil {
		return err
	}

	s.Proof.Score = make([]byte, 32)
	if err := encoding.Read256(b, s.Proof.Score); err != nil {
		return err
	}

	s.Proof.Z = make([]byte, 32)
	if err := encoding.Read256(b, s.Proof.Z); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(b, &s.Proof.BinaryBidList); err != nil {
		return err
	}

	s.Seed = make([]byte, 33)
	return encoding.ReadBLS(b, s.Seed)
}
