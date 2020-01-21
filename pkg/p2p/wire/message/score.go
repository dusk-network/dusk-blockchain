package message

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	zkproof "github.com/dusk-network/dusk-zkproof"
)

type (

	// ScoreProposal is an internal packet created by the node. the Score Message with the fields consistent with the Blind Bid data structure
	ScoreProposal struct {
		hdr           header.Header
		Score         []byte
		Proof         []byte
		Z             []byte
		BidListSubset []byte
		Seed          []byte
	}

	// Score extends the ScoreProposal with additional fields related to the
	// candidate it pairs up with. The Score is supposed to be immutable once
	// created and it gets forwarded to the other nodes
	Score struct {
		ScoreProposal
		PrevHash []byte
		VoteHash []byte
	}
)

func EmptyScore(hdr header.Header) ScoreProposal {
	return ScoreProposal{
		hdr: hdr,
	}
}

// NewScoreProposal creates a new ScoreProposa
func NewScoreProposal(hdr header.Header, seed []byte, proof zkproof.ZkProof) ScoreProposal {
	return ScoreProposal{
		hdr:           hdr,
		Score:         proof.Score,
		Proof:         proof.Proof,
		Z:             proof.Z,
		BidListSubset: proof.BinaryBidList,
		Seed:          seed,
	}
}

// Header is used to comply to the consensus.Message interface
func (e ScoreProposal) State() header.Header {
	return e.hdr
}

// Sender of a Score event is the anonymous Z
func (e ScoreProposal) Sender() []byte {
	return e.Z
}

// NewScore creates a new Score from a proposal
func NewScore(proposal ScoreProposal, pubkey, prevHash, voteHash []byte) *Score {
	score := &Score{
		ScoreProposal: proposal,
		PrevHash:      prevHash,
		VoteHash:      voteHash,
	}
	score.ScoreProposal.hdr.PubKeyBLS = pubkey
	score.ScoreProposal.hdr.BlockHash = voteHash
	return score
}

func (e Score) Equal(s Score) bool {
	return e.hdr.Equal(s.hdr) && bytes.Equal(e.VoteHash, s.VoteHash)
}

func makeScore() *Score {
	return &Score{
		ScoreProposal: ScoreProposal{
			hdr: header.Header{},
		},
	}
}

func UnmarshalScoreMessage(r *bytes.Buffer, m SerializableMessage) error {
	sc := makeScore()

	if err := UnmarshalScore(r, sc); err != nil {
		return err
	}
	m.SetPayload(sc)
	return nil
}

// UnmarshalScore unmarshals the buffer into a Score Event
// Field order is the following:
// * Score Payload [score, proof, Z, BidList, Seed, Block Candidate Hash]
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
func MarshalScore(r *bytes.Buffer, sev Score) error {
	// Marshalling header first
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

func MockScoreProposal(hdr header.Header) ScoreProposal {
	score, _ := crypto.RandEntropy(32)
	proof, _ := crypto.RandEntropy(1477)
	z, _ := crypto.RandEntropy(32)
	subset, _ := crypto.RandEntropy(32)
	seed, _ := crypto.RandEntropy(33)
	return ScoreProposal{
		hdr:           hdr,
		Score:         score,
		Proof:         proof,
		Z:             z,
		BidListSubset: subset,
		Seed:          seed,
	}
}

// MockScore mocks a Score and returns it.
func MockScore(hdr header.Header, hash []byte) Score {
	prevHash, _ := crypto.RandEntropy(32)

	return Score{
		ScoreProposal: MockScoreProposal(hdr),
		PrevHash:      prevHash,
		VoteHash:      hash,
	}
}
