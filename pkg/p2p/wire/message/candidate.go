package message

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
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
// It includes the certificates of the previous block, so the generator can use
// it in the Distribute transaction to reward those provisioners
func MakeCandidate(blk *block.Block, cert *block.Certificate) Candidate {
	c := NewCandidate()
	c.Block = blk
	c.Certificate = cert
	return *c
}

// Copy complies with the message.Safe interface. It performs a DeepCopy
// that can be handy when publishing this Payload for multiple subscribers to
// consume
func (c Candidate) Copy() payload.Safe {
	r := Candidate{}

	if c.Block != nil {
		blk := c.Block.Copy().(block.Block)
		r.Block = &blk
	}

	if c.Certificate != nil {
		r.Certificate = c.Certificate.Copy()
	}

	return r
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
	_, _ = sb.WriteString(c.State().String())
	_, _ = sb.WriteString(" nr. of tx in the Block='")
	_, _ = sb.WriteString(strconv.Itoa(len(c.Block.Txs)))
	if c.Certificate.Step == 0 {
		_, _ = sb.WriteString("' certificate='<empty>'")
	} else {
		_, _ = sb.WriteString("' certificate step='")
		_, _ = sb.WriteString(strconv.FormatUint(uint64(c.Certificate.Step), 10))
		_, _ = sb.WriteString("' certificate signature first step='")
		_, _ = sb.WriteString(util.StringifyBytes(c.Certificate.StepOneBatchedSig))
		_, _ = sb.WriteString("' certificate signature second step='")
		_, _ = sb.WriteString(util.StringifyBytes(c.Certificate.StepTwoBatchedSig))
		_, _ = sb.WriteString("'")
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

// MockCertificate mocks a certificate
func MockCertificate(hash []byte, round uint64, keys []key.Keys, p *user.Provisioners) *block.Certificate {
	votes := GenVotes(hash, round, 3, keys, p)
	return &block.Certificate{
		StepOneBatchedSig: votes[0].Signature.Compress(),
		StepTwoBatchedSig: votes[1].Signature.Compress(),
		Step:              1,
		StepOneCommittee:  votes[0].BitSet,
		StepTwoCommittee:  votes[1].BitSet,
	}
}
