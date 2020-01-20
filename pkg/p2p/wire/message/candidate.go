package message

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-wallet/block"
)

type Candidate struct {
	hdr header.Header
	*block.Block
	*block.Certificate
}

func NewCandidate() *Candidate {
	return &Candidate{
		hdr:         header.Header{},
		Block:       block.NewBlock(),
		Certificate: block.EmptyCertificate(),
	}
}

// Header is for complying to the consensus.Message interface
func (c Candidate) State() header.Header {
	return c.hdr
}

// Sender identifies the sender of a Candidate block
func (c Candidate) Sender() []byte {
	return c.hdr.Sender()
}

func (c Candidate) Equal(m Message) bool {
	can, ok := m.Payload().(Candidate)
	return ok && c.hdr.Equal(can.hdr)
}

func UnmarshalCandidateMessage(b *bytes.Buffer, m SerializableMessage) error {
	cm := NewCandidate()
	if err := header.Unmarshal(b, &cm.hdr); err != nil {
		return err
	}

	if err := UnmarshalCandidate(b, cm); err != nil {
		return err
	}

	m.SetPayload(cm)
	return nil
}

func UnmarshalCandidate(b *bytes.Buffer, cMsg *Candidate) error {
	if err := UnmarshalBlock(b, cMsg.Block); err != nil {
		return err
	}

	return UnmarshalCertificate(b, cMsg.Certificate)
}

func MarshalCandidate(b *bytes.Buffer, cm Candidate) error {
	if err := header.Marshal(b, cm.State()); err != nil {
		return err
	}

	if err := MarshalBlock(b, cm.Block); err != nil {
		return err
	}

	return MarshalCertificate(b, cm.Certificate)
}
