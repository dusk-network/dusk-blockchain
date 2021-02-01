// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package message

import (
	"bytes"
	"math/big"
	"strings"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/blindbid"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

type (
	// ScoreProposal is an internal packet created by the node. the Score Message with the fields consistent with the Blind Bid data structure.
	ScoreProposal struct {
		hdr      header.Header
		Score    []byte
		Proof    []byte
		Identity []byte
		Seed     []byte
	}

	// Score extends the ScoreProposal with additional fields related to the
	// candidate it pairs up with. The Score is supposed to be immutable once
	// created and it gets forwarded to the other nodes.
	Score struct {
		ScoreProposal
		PrevHash  []byte
		Candidate block.Block
	}
)

// EmptyScoreProposal is used to initialize a ScoreProposal. It is used
// primarily by the internal Score generator.
func EmptyScoreProposal(hdr header.Header) ScoreProposal {
	return ScoreProposal{
		hdr: hdr,
	}
}

// NewScoreProposal creates a new ScoreProposal.
func NewScoreProposal(hdr header.Header, seed []byte, score blindbid.GenerateScoreResponse) ScoreProposal {
	return ScoreProposal{
		hdr:      hdr,
		Score:    score.Score,
		Proof:    score.BlindbidProof,
		Seed:     seed,
		Identity: score.ProverIdentity,
	}
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (e ScoreProposal) Copy() payload.Safe {
	cpy := ScoreProposal{
		hdr: e.hdr.Copy().(header.Header),
	}

	if e.Score != nil {
		cpy.Score = make([]byte, len(e.Score))
		copy(cpy.Score, e.Score)
	}

	if e.Proof != nil {
		cpy.Proof = make([]byte, len(e.Proof))
		copy(cpy.Proof, e.Proof)
	}

	if e.Seed != nil {
		cpy.Seed = make([]byte, len(e.Seed))
		copy(cpy.Seed, e.Seed)
	}

	if e.Identity != nil {
		cpy.Identity = make([]byte, len(e.Identity))
		copy(cpy.Identity, e.Identity)
	}

	return cpy
}

// IsEmpty tests a ScoreProposal for emptyness.
func (e ScoreProposal) IsEmpty() bool {
	return e.Score == nil
}

// State is used to comply to the consensus.Message interface.
func (e ScoreProposal) State() header.Header {
	return e.hdr
}

// Sender of a Score event is the anonymous Z.
func (e ScoreProposal) Sender() []byte {
	return e.Identity
}

// String representation of the ScoreProposal.
func (e ScoreProposal) String() string {
	var sb strings.Builder

	_, _ = sb.WriteString(e.hdr.String())
	_, _ = sb.WriteString(" score='")
	_, _ = sb.WriteString(util.StringifyBytes(e.Score))
	_, _ = sb.WriteString(" seed='")
	_, _ = sb.WriteString(util.StringifyBytes(e.Seed))

	return sb.String()
}

// NewScore creates a new Score from a proposal.
func NewScore(proposal ScoreProposal, pubkey, prevHash []byte, candidate block.Block) *Score {
	score := &Score{
		ScoreProposal: proposal,
		PrevHash:      prevHash,
		Candidate:     candidate,
	}

	score.ScoreProposal.hdr.PubKeyBLS = pubkey
	score.ScoreProposal.hdr.BlockHash = score.VoteHash()
	return score
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (e Score) Copy() payload.Safe {
	cpy := Score{
		ScoreProposal: e.ScoreProposal.Copy().(ScoreProposal),
		PrevHash:      make([]byte, len(e.PrevHash)),
		Candidate:     e.Candidate.Copy().(block.Block),
	}

	copy(cpy.PrevHash, e.PrevHash)
	return cpy
}

// EmptyScore is used primarily to initialize the Score,
// since empty scores should not be propagated externally.
func EmptyScore() Score {
	return Score{
		ScoreProposal: EmptyScoreProposal(header.Header{BlockHash: make([]byte, 32)}),
		Candidate:     *block.NewBlock(),
	}
}

// Equal tests if two Scores are equal.
func (e Score) Equal(s Score) bool {
	return e.hdr.Equal(s.hdr) && bytes.Equal(e.VoteHash(), s.VoteHash()) && e.VoteHash() != nil
}

// VoteHash returns hash of the Candidate block.
func (e Score) VoteHash() []byte {
	return e.Candidate.Header.Hash
}

// String representation of a Score.
func (e Score) String() string {
	var sb strings.Builder

	_, _ = sb.WriteString(e.ScoreProposal.String())
	_, _ = sb.WriteString(" prev_hash='")
	_, _ = sb.WriteString(util.StringifyBytes(e.PrevHash))
	_, _ = sb.WriteString(" vote_hash='")
	_, _ = sb.WriteString(util.StringifyBytes(e.VoteHash()))

	return sb.String()
}

func makeScore() *Score {
	return &Score{
		ScoreProposal: ScoreProposal{
			hdr: header.Header{},
		},
	}
}

// UnmarshalScoreMessage unmarshal a ScoreMessage from a buffer.
func UnmarshalScoreMessage(r *bytes.Buffer, m SerializableMessage) error {
	sc := makeScore()

	if err := UnmarshalScore(r, sc); err != nil {
		return err
	}

	m.SetPayload(*sc)
	return nil
}

// UnmarshalScore unmarshals the buffer into a Score Event.
// Field order is the following:
// * Score Payload [score, proof, Z, BidList, Seed, Block Candidate Hash].
func UnmarshalScore(r *bytes.Buffer, sev *Score) error {
	if err := header.Unmarshal(r, &sev.hdr); err != nil {
		return err
	}

	sev.Score = make([]byte, 32)
	if err := encoding.Read256(r, sev.Score); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &sev.Proof); err != nil {
		return err
	}

	sev.Identity = make([]byte, 32)
	if err := encoding.Read256(r, sev.Identity); err != nil {
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

	sev.Candidate = *block.NewBlock()
	if err := UnmarshalBlock(r, &sev.Candidate); err != nil {
		return err
	}

	return nil
}

// MarshalScore the buffer into a committee Event.
// Field order is the following:
// * Blind Bid Fields [Score, Proof, Z, BidList, Seed, Candidate Block Hash].
func MarshalScore(r *bytes.Buffer, sev Score) error {
	// Marshaling header first
	if err := header.Marshal(r, sev.hdr); err != nil {
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

	// Identity
	if err := encoding.Write256(r, sev.Identity); err != nil {
		return err
	}

	if err := encoding.Write256(r, sev.PrevHash); err != nil {
		return err
	}

	// Seed
	if err := encoding.WriteBLS(r, sev.Seed); err != nil {
		return err
	}

	// Candidate
	if err := MarshalBlock(r, &sev.Candidate); err != nil {
		return err
	}

	return nil
}

// MockScoreProposal mocks a ScoreProposal up.
func MockScoreProposal(hdr header.Header) ScoreProposal {
	var score []byte

	for {
		score, _ = crypto.RandEntropy(32)
		scoreInt := big.NewInt(0).SetBytes(score)
		limit, _ := big.NewInt(0).SetString("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", 16)

		if scoreInt.Cmp(limit) > -1 {
			break
		}
	}

	proof, _ := crypto.RandEntropy(1477)
	identity, _ := crypto.RandEntropy(32)
	seed, _ := crypto.RandEntropy(33)

	return ScoreProposal{
		hdr:      hdr,
		Score:    score,
		Proof:    proof,
		Identity: identity,
		Seed:     seed,
	}
}

// MockScore mocks a Score and returns it.
func MockScore(hdr header.Header, c block.Block) Score {
	prevHash, _ := crypto.RandEntropy(32)

	return Score{
		ScoreProposal: MockScoreProposal(hdr),
		PrevHash:      prevHash,
		Candidate:     c,
	}
}
