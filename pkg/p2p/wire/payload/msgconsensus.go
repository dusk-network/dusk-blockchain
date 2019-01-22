package payload

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// MsgConsensus defines a consensus message on the Dusk wire protocol.
// The message can contain any consensus-related payload, identified by
// the ID field.
type MsgConsensus struct {
	Version       uint32           // Node version
	Round         uint64           // The round this consensus message is for
	PrevBlockHash []byte           // Hash of the previous block header
	ID            consensusmsg.ID  // Payload identifier
	Payload       consensusmsg.Msg // Consensus payload
	Signature     []byte           // Ed25519 signature of all above fields
	PubKey        []byte           // Ed25519 public key of the sender
}

// NewMsgConsensus will return a MsgConsensus struct populated with the passed
// parameters. The payload ID is inferred from the content. The function will
// do some sanity checks on byte slice lengths and throw an error in event of
// a mismatch.
func NewMsgConsensus(version uint32, round uint64, prevBlockHash, sig, pk []byte,
	pl consensusmsg.Msg) (*MsgConsensus, error) {
	if len(prevBlockHash) != 32 {
		return nil, errors.New("wire: supplied previous block hash for consensus message is improper length")
	}

	if len(sig) != 64 {
		return nil, errors.New("wire: supplied ed25519 signature for consensus message is improper length")
	}

	if len(pk) != 32 {
		return nil, errors.New("wire: supplied ed25519 public key for consensus message is improper length")
	}

	return &MsgConsensus{
		Version:       version,
		Round:         round,
		PrevBlockHash: prevBlockHash,
		ID:            pl.Type(),
		Payload:       pl,
		Signature:     sig,
		PubKey:        pk,
	}, nil
}

// Signable returns the fields of MsgConsensus that were signed by the
// sender as a byte slice for easy verification during consensus.
func (m *MsgConsensus) Signable() ([]byte, error) {
	w := new(bytes.Buffer)
	if err := encoding.WriteUint32(w, binary.LittleEndian, m.Version); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, m.Round); err != nil {
		return nil, err
	}

	if err := encoding.Write256(w, m.PrevBlockHash); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint8(w, uint8(m.ID)); err != nil {
		return nil, err
	}

	if err := m.Payload.Encode(w); err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// Encode a MsgConsensus to w
// Implements Payload interface
func (m *MsgConsensus) Encode(w io.Writer) error {
	if err := encoding.WriteUint32(w, binary.LittleEndian, m.Version); err != nil {
		return err
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, m.Round); err != nil {
		return err
	}

	if err := encoding.Write256(w, m.PrevBlockHash); err != nil {
		return err
	}

	if err := encoding.WriteUint8(w, uint8(m.ID)); err != nil {
		return err
	}

	if err := m.Payload.Encode(w); err != nil {
		return err
	}

	if err := encoding.Write512(w, m.Signature); err != nil {
		return err
	}

	if err := encoding.Write256(w, m.PubKey); err != nil {
		return err
	}

	return nil
}

// Decode a MsgConsensus from r
// Implements Payload interface
func (m *MsgConsensus) Decode(r io.Reader) error {
	if err := encoding.ReadUint32(r, binary.LittleEndian, &m.Version); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &m.Round); err != nil {
		return err
	}

	if err := encoding.Read256(r, &m.PrevBlockHash); err != nil {
		return err
	}

	var ID uint8
	if err := encoding.ReadUint8(r, &ID); err != nil {
		return err
	}

	m.ID = consensusmsg.ID(ID)
	var pl consensusmsg.Msg
	switch m.ID {
	case consensusmsg.CandidateScoreID:
		p := &consensusmsg.CandidateScore{}
		if err := p.Decode(r); err != nil {
			return err
		}

		pl = p
	case consensusmsg.CandidateID:
		p := &consensusmsg.Candidate{}
		if err := p.Decode(r); err != nil {
			return err
		}

		pl = p
	case consensusmsg.ReductionID:
		p := &consensusmsg.Reduction{}
		if err := p.Decode(r); err != nil {
			return err
		}

		pl = p
	case consensusmsg.AgreementID:
		p := &consensusmsg.Agreement{}
		if err := p.Decode(r); err != nil {
			return err
		}

		pl = p
	/*case consensusmsg.SetAgreementID:
	p := &consensusmsg.SetAgreement{}
	if err := p.Decode(r); err != nil {
		return err
	}

	pl = p*/
	case consensusmsg.SigSetCandidateID:
		p := &consensusmsg.SigSetCandidate{}
		if err := p.Decode(r); err != nil {
			return err
		}

		pl = p
	case consensusmsg.SigSetVoteID:
		p := &consensusmsg.SigSetVote{}
		if err := p.Decode(r); err != nil {
			return err
		}

		pl = p
	default:
		return fmt.Errorf("wire: consensus payload has unrecognized ID %v", ID)
	}

	m.Payload = pl
	if err := encoding.Read512(r, &m.Signature); err != nil {
		return err
	}

	if err := encoding.Read256(r, &m.PubKey); err != nil {
		return err
	}

	return nil
}

// Command returns the command string associated with the consensus message.
// Implements Payload interface
func (m *MsgConsensus) Command() commands.Cmd {
	return commands.Consensus
}
