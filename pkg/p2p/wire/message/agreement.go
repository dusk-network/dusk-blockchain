package message

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
	"github.com/dusk-network/dusk-crypto/bls"
	"github.com/dusk-network/dusk-wallet/v2/block"
	"github.com/dusk-network/dusk-wallet/v2/key"
)

type (
	// StepVotes represents the aggregated votes for one reduction step.
	// Normally an Agreement event includes two of these structures. They need to
	// be kept separated since the BitSet representation of the Signees does not
	// admit duplicates, whereas the same provisioner may very well be included in
	// the committee for both Reduction steps
	StepVotes struct {
		Apk       *bls.Apk
		BitSet    uint64
		Signature *bls.Signature
		Step      uint8
	}

	// StepVotesMsg is the internal message exchanged by the consensus
	// components (through the signer.SendInternally method). It is not meant for
	// external communications and therefore it does not have a
	// Marshal/Unmarshal methods associated
	StepVotesMsg struct {
		hdr header.Header
		StepVotes
	}

	// Agreement is the Event created at the end of the Reduction process. It includes
	// the aggregated compressed signatures of all voters
	Agreement struct {
		hdr          header.Header
		signedVotes  []byte
		VotesPerStep []*StepVotes
		Repr         *big.Int
	}
)

// String representation of the Agreement
func (a Agreement) String() string {
	var sb strings.Builder
	sb.WriteString(a.hdr.String())
	sb.WriteString(" signature='")
	sb.WriteString(util.StringifyBytes(a.signedVotes))
	sb.WriteString(" repr='")
	sb.WriteString(util.StringifyBytes(a.Repr.Bytes()))
	return sb.String()
}

func NewStepVotesMsg(round uint64, hash []byte, sender []byte, sv StepVotes) StepVotesMsg {
	return StepVotesMsg{
		hdr: header.Header{
			Step:      sv.Step,
			Round:     round,
			BlockHash: hash,
			PubKeyBLS: sender,
		},
		StepVotes: sv,
	}
}

// State returns the Header without information about Sender (as this is only
// for internal communications)
func (s StepVotesMsg) State() header.Header {
	return s.hdr
}

// IsEmpty returns whether the StepVotesMsg represents a failed convergence
// attempt at consensus over a Reduction message
func (s StepVotes) IsEmpty() bool {
	return s.Apk == nil
}

// String representation
func (s StepVotes) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("BitSet: %d Step: %d\n Sig: %v\n Apk: %v\n", s.BitSet, s.Step, s.Signature, s.Apk))
	return sb.String()
}

// Header returns the message header. This is to comply to the
// consensus.Message interface
func (a Agreement) State() header.Header {
	return a.hdr
}

func (a Agreement) Sender() []byte {
	return a.hdr.Sender()
}

func (a Agreement) Cmp(other Agreement) int {
	return a.Repr.Cmp(other.Repr)
}

func (a *Agreement) SetSignature(signedVotes []byte) {
	a.Repr = new(big.Int).SetBytes(signedVotes)
	a.signedVotes = signedVotes
}

func (a Agreement) SignedVotes() []byte {
	return a.signedVotes
}

func (a Agreement) Equal(aev Agreement) bool {
	return a.Repr.Cmp(aev.Repr) == 0
}

// GenerateCertificate is used by the Chain component
func (a Agreement) GenerateCertificate() *block.Certificate {
	return &block.Certificate{
		StepOneBatchedSig: a.VotesPerStep[0].Signature.Compress(),
		StepTwoBatchedSig: a.VotesPerStep[1].Signature.Compress(),
		Step:              a.State().Step,
		StepOneCommittee:  a.VotesPerStep[0].BitSet,
		StepTwoCommittee:  a.VotesPerStep[1].BitSet,
	}
}

// UnmarshalAgreementMessage unmarshal a network inbound Agreement
func UnmarshalAgreementMessage(r *bytes.Buffer, m SerializableMessage) error {
	aggro := newAgreement()
	if err := header.Unmarshal(r, &aggro.hdr); err != nil {
		return err
	}

	if err := UnmarshalAgreement(r, aggro); err != nil {
		return err
	}

	m.SetPayload(*aggro)
	return nil
}

// NewStepVotes returns a new StepVotes structure for a given round, step and block hash
func NewStepVotes() *StepVotes {
	return &StepVotes{
		Apk:       nil,
		BitSet:    uint64(0),
		Signature: nil,
		Step:      uint8(0),
	}
}

// Equal checks if two StepVotes structs are the same.
func (sv *StepVotes) Equal(other *StepVotes) bool {
	return bytes.Equal(sv.Apk.Marshal(), other.Apk.Marshal()) &&
		bytes.Equal(sv.Signature.Marshal(), other.Signature.Marshal())
}

// Add a vote to the StepVotes struct.
func (sv *StepVotes) Add(signature, sender []byte, step uint8) error {
	if sv.Apk == nil {
		pk, err := bls.UnmarshalPk(sender)
		if err != nil {
			return err
		}
		sv.Step = step
		sv.Apk = bls.NewApk(pk)
		sv.Signature, err = bls.UnmarshalSignature(signature)
		if err != nil {
			return err
		}

		return nil
	}

	if step != sv.Step {
		return fmt.Errorf("mismatched step in aggregating vote set. Expected %d, got %d", sv.Step, step)
	}

	if err := sv.Apk.AggregateBytes(sender); err != nil {
		return err
	}
	if err := sv.Signature.AggregateBytes(signature); err != nil {
		return err
	}

	return nil
}

// MarshalAgreement marshals an Agreement event into a buffer.
func MarshalAgreement(r *bytes.Buffer, a Agreement) error {
	if err := header.Marshal(r, a.State()); err != nil {
		return err
	}

	// Marshal BLS Signature of VoteSet
	if err := encoding.WriteBLS(r, a.SignedVotes()); err != nil {
		return err
	}

	// Marshal VotesPerStep
	if err := MarshalVotes(r, a.VotesPerStep); err != nil {
		return err
	}

	return nil
}

// Unmarshal unmarshals the buffer into an Agreement
// Field order is the following:
// * Header [BLS Public Key; Round; Step]
// * Agreement [Signed Vote Set; Vote Set; BlockHash]
func UnmarshalAgreement(r *bytes.Buffer, a *Agreement) error {
	signedVotes := make([]byte, 33)
	if err := encoding.ReadBLS(r, signedVotes); err != nil {
		return err
	}
	a.SetSignature(signedVotes)

	votesPerStep := make([]*StepVotes, 2)
	if err := UnmarshalVotes(r, votesPerStep); err != nil {
		return err
	}

	a.VotesPerStep = votesPerStep
	return nil
}

// NewAgreement returns an empty Agreement event. It is supposed to be used by
// the (secondstep reducer) for creating Agreement messages
func NewAgreement(hdr header.Header) *Agreement {
	aggro := newAgreement()
	aggro.hdr = hdr
	return aggro
}

// newAgreement returns an empty Agreement event. It is used within the
// UnmarshalAgreement function
// TODO: interface - []*StepVotes should not be references, but values
func newAgreement() *Agreement {
	return &Agreement{
		hdr:          header.Header{},
		VotesPerStep: make([]*StepVotes, 2),
		signedVotes:  make([]byte, 33),
		Repr:         new(big.Int),
	}
}

// SignAgreement signs an aggregated agreement event
// XXX: either use this function or delete it!! Right now it is not used
func SignAgreement(a *Agreement, keys key.ConsensusKeys) error {
	buffer := new(bytes.Buffer)
	if err := MarshalVotes(buffer, a.VotesPerStep); err != nil {
		return err
	}

	signedVoteSet, err := bls.Sign(keys.BLSSecretKey, keys.BLSPubKey, buffer.Bytes())
	if err != nil {
		return err
	}

	a.SetSignature(signedVoteSet.Compress())
	return nil
}

// UnmarshalVotes unmarshals the array of StepVotes for a single Agreement
func UnmarshalVotes(r *bytes.Buffer, votes []*StepVotes) error {
	length, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	// Agreement can only ever have two StepVotes, for the two
	// reduction steps.
	if length != 2 {
		return errors.New("malformed Agreement message")
	}

	for i := uint64(0); i < length; i++ {
		sv, err := UnmarshalStepVotes(r)
		if err != nil {
			return err
		}

		votes[i] = sv
	}

	return nil
}

// UnmarshalStepVotes unmarshals a single StepVote
func UnmarshalStepVotes(r *bytes.Buffer) (*StepVotes, error) {
	sv := NewStepVotes()
	// APK
	var apk []byte
	if err := encoding.ReadVarBytes(r, &apk); err != nil {
		return nil, err
	}

	var err error
	sv.Apk, err = bls.UnmarshalApk(apk)
	if err != nil {
		return nil, err
	}

	// BitSet
	if err := encoding.ReadUint64LE(r, &sv.BitSet); err != nil {
		return nil, err
	}

	// Signature
	signature := make([]byte, 33)
	if err := encoding.ReadBLS(r, signature); err != nil {
		return nil, err
	}

	sv.Signature, err = bls.UnmarshalSignature(signature)
	if err != nil {
		return nil, err
	}

	return sv, nil
}

// MarshalVotes marshals an array of StepVotes
func MarshalVotes(r *bytes.Buffer, votes []*StepVotes) error {
	if err := encoding.WriteVarInt(r, uint64(len(votes))); err != nil {
		return err
	}

	for _, stepVotes := range votes {
		if err := MarshalStepVotes(r, stepVotes); err != nil {
			return err
		}
	}

	return nil
}

// MarshalStepVotes marshals the aggregated form of the BLS PublicKey and Signature
// for an ordered set of votes
func MarshalStepVotes(r *bytes.Buffer, vote *StepVotes) error {
	// APK
	if err := encoding.WriteVarBytes(r, vote.Apk.Marshal()); err != nil {
		return err
	}

	// BitSet
	if err := encoding.WriteUint64LE(r, vote.BitSet); err != nil {
		return err
	}

	// Signature
	if err := encoding.WriteBLS(r, vote.Signature.Compress()); err != nil {
		return err
	}
	return nil
}

// MockAgreement returns a mocked Agreement Event, to be used for testing purposes.
// It includes a vararg iterativeIdx to help avoiding duplicates when testing
func MockAgreement(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys, p *user.Provisioners, iterativeIdx ...int) Agreement {
	// Make sure we create an event made by an actual voting committee member
	c := p.CreateVotingCommittee(round, step, len(keys))
	cKeys := createCommitteeKeySet(c, keys)

	idx := 0
	if len(iterativeIdx) != 0 {
		idx = iterativeIdx[0]
	}

	if idx > len(keys) {
		panic("wrong iterative index: cannot iterate more than there are keys")
	}

	hdr := header.Header{Round: round, Step: step, BlockHash: hash, PubKeyBLS: cKeys[idx].BLSPubKeyBytes}
	a := NewAgreement(hdr)

	// generating reduction events (votes) and signing them
	steps := GenVotes(hash, round, step, keys, p)

	whole := new(bytes.Buffer)
	if err := header.MarshalSignableVote(whole, a.State()); err != nil {
		panic(err)
	}

	sig, _ := bls.Sign(cKeys[idx].BLSSecretKey, cKeys[idx].BLSPubKey, whole.Bytes())
	a.VotesPerStep = steps
	a.SetSignature(sig.Compress())
	return *a
}

func MockCommitteeVoteSet(p *user.Provisioners, k []key.ConsensusKeys, hash []byte, committeeSize int, round uint64, step uint8) []Reduction {
	c1 := p.CreateVotingCommittee(round, step-2, len(k))
	c2 := p.CreateVotingCommittee(round, step-1, len(k))
	cKeys1 := createCommitteeKeySet(c1, k)
	cKeys2 := createCommitteeKeySet(c2, k)
	events := createVoteSet(cKeys1, cKeys2, hash, len(cKeys1), round, step)
	return events
}

// GenVotes randomly generates a slice of StepVotes with the indicated lenght.
// Albeit random, the generation is consistent with the rules of Votes
func GenVotes(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys, p *user.Provisioners) []*StepVotes {
	if len(keys) < 2 {
		panic("At least two votes are required to mock an Agreement")
	}

	// Create committee key sets
	keySet1 := createCommitteeKeySet(p.CreateVotingCommittee(round, step-2, len(keys)), keys)
	keySet2 := createCommitteeKeySet(p.CreateVotingCommittee(round, step-1, len(keys)), keys)

	stepVotes1, set1 := createStepVotesAndSet(hash, round, step-2, keySet1)
	stepVotes2, set2 := createStepVotesAndSet(hash, round, step-1, keySet2)

	bitSet1 := createBitSet(set1, round, step-2, len(keySet1), p)
	stepVotes1.BitSet = bitSet1
	bitSet2 := createBitSet(set2, round, step-1, len(keySet2), p)
	stepVotes2.BitSet = bitSet2

	return []*StepVotes{stepVotes1, stepVotes2}
}

func createBitSet(set sortedset.Set, round uint64, step uint8, size int, p *user.Provisioners) uint64 {
	committee := p.CreateVotingCommittee(round, step, size)
	return committee.Bits(set)
}

func createCommitteeKeySet(c user.VotingCommittee, k []key.ConsensusKeys) (keys []key.ConsensusKeys) {
	committeeKeys := c.MemberKeys()

	for _, cKey := range committeeKeys {
		for _, key := range k {
			if bytes.Equal(cKey, key.BLSPubKeyBytes) {
				keys = append(keys, key)
				break
			}
		}
	}

	return keys
}

func createStepVotesAndSet(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys) (*StepVotes, sortedset.Set) {
	set := sortedset.New()
	stepVotes := NewStepVotes()
	for _, k := range keys {

		// We should not aggregate any given key more than once.
		_, inserted := set.IndexOf(k.BLSPubKeyBytes)
		if !inserted {
			h := header.Header{
				BlockHash: hash,
				Round:     round,
				Step:      step,
				PubKeyBLS: k.BLSPubKeyBytes,
			}

			r := new(bytes.Buffer)
			if err := header.MarshalSignableVote(r, h); err != nil {
				panic(err)
			}

			sigma, _ := bls.Sign(k.BLSSecretKey, k.BLSPubKey, r.Bytes())
			if err := stepVotes.Add(sigma.Compress(), k.BLSPubKeyBytes, step); err != nil {
				panic(err)
			}
		}

		set.Insert(k.BLSPubKeyBytes)
	}

	return stepVotes, set
}
