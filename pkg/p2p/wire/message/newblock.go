// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package message

import (
	"bytes"
	"strings"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

// NewBlock extends the ScoreProposal with additional fields related to the
// candidate it pairs up with. The NewBlock is supposed to be immutable once
// created and it gets forwarded to the other nodes.
type NewBlock struct {
	hdr        header.Header
	PrevHash   []byte
	Candidate  block.Block
	SignedHash []byte
}

// NewNewBlock creates a new Score from a proposal.
func NewNewBlock(hdr header.Header, prevHash []byte, candidate block.Block) *NewBlock {
	return &NewBlock{
		hdr:        hdr,
		PrevHash:   prevHash,
		Candidate:  candidate,
		SignedHash: make([]byte, 0),
	}
}

// State is used to comply to the consensus.Message interface.
func (e NewBlock) State() header.Header {
	return e.hdr
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (e NewBlock) Copy() payload.Safe {
	cpy := NewBlock{
		hdr:        e.hdr.Copy().(header.Header),
		PrevHash:   make([]byte, len(e.PrevHash)),
		Candidate:  e.Candidate.Copy().(block.Block),
		SignedHash: make([]byte, len(e.SignedHash)),
	}

	copy(cpy.PrevHash, e.PrevHash)
	copy(cpy.SignedHash, e.SignedHash)
	return cpy
}

// IsEmpty checks if a NewBlock message is empty.
func (e NewBlock) IsEmpty() bool {
	return e.SignedHash == nil && e.PrevHash == nil
}

// EmptyNewBlock is used primarily to initialize the NewBlock,
// since empty newblocks should not be propagated externally.
func EmptyNewBlock() NewBlock {
	return NewBlock{
		hdr:       header.New(),
		Candidate: *block.NewBlock(),
	}
}

// Equal tests if two NewBlock messages are equal.
func (e NewBlock) Equal(s NewBlock) bool {
	return e.hdr.Equal(s.hdr) && bytes.Equal(e.VoteHash(), s.VoteHash()) && e.VoteHash() != nil
}

// VoteHash returns hash of the Candidate block.
func (e NewBlock) VoteHash() []byte {
	return e.Candidate.Header.Hash
}

// String representation of a NewBlock.
func (e NewBlock) String() string {
	var sb strings.Builder

	_, _ = sb.WriteString(e.hdr.String())
	_, _ = sb.WriteString(" prev_hash='")
	_, _ = sb.WriteString(util.StringifyBytes(e.PrevHash))
	_, _ = sb.WriteString(" vote_hash='")
	_, _ = sb.WriteString(util.StringifyBytes(e.VoteHash()))
	_, _ = sb.WriteString(" signed_hash='")
	_, _ = sb.WriteString(util.StringifyBytes(e.SignedHash))

	return sb.String()
}

func makeNewBlock() *NewBlock {
	return &NewBlock{
		hdr: header.Header{},
	}
}

// UnmarshalNewBlockMessage unmarshal a NewBlockMessage from a buffer.
func UnmarshalNewBlockMessage(r *bytes.Buffer, m SerializableMessage) error {
	sc := makeNewBlock()

	if err := UnmarshalNewBlock(r, sc); err != nil {
		return err
	}

	m.SetPayload(*sc)
	return nil
}

// UnmarshalNewBlock unmarshals the buffer into a NewBlock Event.
// Field order is the following:
// * Score Payload [score, proof, Z, BidList, Seed, Block Candidate Hash].
func UnmarshalNewBlock(r *bytes.Buffer, sev *NewBlock) error {
	if err := header.Unmarshal(r, &sev.hdr); err != nil {
		return err
	}

	sev.PrevHash = make([]byte, 32)
	if err := encoding.Read256(r, sev.PrevHash); err != nil {
		return err
	}

	sev.Candidate = *block.NewBlock()
	if err := UnmarshalBlock(r, &sev.Candidate); err != nil {
		return err
	}

	sev.SignedHash = make([]byte, 0)
	if err := encoding.ReadVarBytes(r, &sev.SignedHash); err != nil {
		return err
	}

	return nil
}

// MarshalNewBlock the buffer into a committee Event.
// Field order is the following:
// * Blind Bid Fields [Score, Proof, Z, BidList, Seed, Candidate Block Hash].
func MarshalNewBlock(r *bytes.Buffer, sev NewBlock) error {
	// Marshaling header first
	if err := header.Marshal(r, sev.hdr); err != nil {
		return err
	}

	if err := encoding.Write256(r, sev.PrevHash); err != nil {
		return err
	}

	// Candidate
	if err := MarshalBlock(r, &sev.Candidate); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, sev.SignedHash); err != nil {
		return err
	}

	return nil
}

// MockNewBlock mocks a NewBlock and returns it.
func MockNewBlock(hdr header.Header, c block.Block) NewBlock {
	prevHash, _ := crypto.RandEntropy(32)

	return NewBlock{
		hdr:        hdr,
		PrevHash:   prevHash,
		Candidate:  c,
		SignedHash: make([]byte, 50),
	}
}
