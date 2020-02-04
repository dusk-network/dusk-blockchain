package message

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
	"github.com/dusk-network/dusk-wallet/v2/block"
	"github.com/dusk-network/dusk-wallet/v2/key"
)

// Candidate is the composition of block and certificates
type Candidate struct {
	*block.Block
	*block.Certificate
}

// NewCandidate is used for instantiating a new Candidate
func NewCandidate() *Candidate {
	return &Candidate{
		Block:       block.NewBlock(),
		Certificate: block.EmptyCertificate(),
	}
}

// MakeCandidate creates a Candidate from a block and a certificate. It is
// meant for actual creation of the Candidate, rather than struct-decoding of a
// transmitted one
func MakeCandidate(blk *block.Block, cert *block.Certificate) Candidate {
	c := NewCandidate()
	c.Block = blk
	c.Certificate = cert
	return *c
}

// State is for complying to the consensus.Message interface. In the case of
// Candidate, the Step does not make sense (since we have one block per round)
// and the Sender is anonymous in nature
func (c Candidate) State() header.Header {
	hdr := header.New()
	hdr.Round = c.Block.Header.Height
	hdr.BlockHash = c.Block.Header.Hash
	hdr.PubKeyBLS = make([]byte, 33)
	hdr.Step = 0
	return hdr
}

// String representation of the Candidate
func (c Candidate) String() string {
	var sb strings.Builder
	sb.WriteString(c.State().String())
	sb.WriteString(" nr. of tx in the Block='")
	sb.WriteString(strconv.Itoa(len(c.Block.Txs)))
	if c.Certificate.Step == 0 {
		sb.WriteString("' certificate='<empty>'")
	} else {
		sb.WriteString("' certificate step='")
		sb.WriteString(strconv.FormatUint(uint64(c.Certificate.Step), 10))
		sb.WriteString("' certificate signature first step='")
		sb.WriteString(util.StringifyBytes(c.Certificate.StepOneBatchedSig))
		sb.WriteString("' certificate signature second step='")
		sb.WriteString(util.StringifyBytes(c.Certificate.StepTwoBatchedSig))
		sb.WriteString("'")
	}
	return sb.String()
}

// Sender is empty (as the BG sends the Candidate in all confidentiality)
func (c Candidate) Sender() []byte {
	return make([]byte, 33)
}

// Equal is needed by the message.Message interface
func (c Candidate) Equal(m Message) bool {
	can, ok := m.Payload().(Candidate)
	return ok && c.State().Equal(can.State())
}

// UnmarshalCandidateMessage encodes a message.Message (with a Candidate
// payload) into a buffer
func UnmarshalCandidateMessage(b *bytes.Buffer, m SerializableMessage) error {
	cm := NewCandidate()
	if err := UnmarshalCandidate(b, cm); err != nil {
		return err
	}

	m.SetPayload(*cm)
	return nil
}

// UnmarshalCandidate consumes a buffer, instantiate and fills the Candidate
// fields
func UnmarshalCandidate(b *bytes.Buffer, c *Candidate) error {
	if err := UnmarshalBlock(b, c.Block); err != nil {
		return err
	}

	return UnmarshalCertificate(b, c.Certificate)
}

// MarshalCandidate encodes a Candidate to a binary form
func MarshalCandidate(b *bytes.Buffer, c Candidate) error {
	if err := MarshalBlock(b, c.Block); err != nil {
		return err
	}

	return MarshalCertificate(b, c.Certificate)
}

func MockCertificate(hash []byte, round uint64, keys []key.ConsensusKeys, p *user.Provisioners) *block.Certificate {
	votes := GenVotes(hash, round, 3, keys, p)
	return &block.Certificate{
		StepOneBatchedSig: votes[0].Signature.Compress(),
		StepTwoBatchedSig: votes[1].Signature.Compress(),
		Step:              1,
		StepOneCommittee:  votes[0].BitSet,
		StepTwoCommittee:  votes[1].BitSet,
	}
}
