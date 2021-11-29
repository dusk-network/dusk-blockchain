// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package message

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/dusk-network/bls12_381-sign/go/cgo/bls"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
)

type (
	// StepVotes represents the aggregated votes for one reduction step.
	// Normally an Agreement event includes two of these structures. They need to
	// be kept separated since the BitSet representation of the Signees does not
	// admit duplicates, whereas the same provisioner may very well be included in
	// the committee for both Reduction steps.
	StepVotes struct {
		BitSet    uint64
		Signature []byte
	}

	// StepVotesMsg is the internal message exchanged by the consensus
	// components (through the signer.SendInternally method). It is not meant for
	// external communications and therefore it does not have a
	// Marshal/Unmarshal methods associated.
	StepVotesMsg struct {
		header.Header
		StepVotes
	}

	// Agreement is the Event created at the end of the Reduction process. It includes
	// the aggregated compressed signatures of all voters.
	Agreement struct {
		hdr          header.Header
		signature    []byte
		VotesPerStep []*StepVotes
		Repr         *big.Int
	}
)

var mockSeed = []byte{0, 0, 0, 0}

// Copy deeply the StepVotes.
func (s *StepVotes) Copy() *StepVotes {
	sig := make([]byte, len(s.Signature))

	copy(sig, s.Signature)

	return &StepVotes{
		BitSet:    s.BitSet,
		Signature: sig,
	}
}

// String representation of the Agreement.
func (a Agreement) String() string {
	var sb strings.Builder

	_, _ = sb.WriteString(a.hdr.String())
	_, _ = sb.WriteString(" signature='")
	_, _ = sb.WriteString(util.StringifyBytes(a.signature))
	_, _ = sb.WriteString(" repr='")
	_, _ = sb.WriteString(util.StringifyBytes(a.Repr.Bytes()))
	_, _ = sb.WriteString(" voteslen='")
	_, _ = sb.WriteString(fmt.Sprintf("%d", len(a.VotesPerStep)))

	return sb.String()
}

// Copy the Agreement is somewhat more expensive than the other structures
// since it involves Marshaling and Unmarshaling. This is necessary since we do
// not have access to the underlying BLS structs.
func (a Agreement) Copy() payload.Safe {
	// NOTE: we ignore the error here. Since we deal with a well formed agreement we
	// assume that the marshaling cannot fail.
	cpy := new(Agreement)
	cpy.hdr = a.hdr.Copy().(header.Header)

	if a.signature != nil {
		cpy.signature = make([]byte, len(a.signature))
		copy(cpy.signature, a.signature)
	}

	cpy.Repr = new(big.Int)
	cpy.Repr.Set(a.Repr)

	if a.VotesPerStep != nil {
		// Un-Marshaling the StepVotes for equality
		cpy.VotesPerStep = make([]*StepVotes, len(a.VotesPerStep))

		for i, vps := range a.VotesPerStep {
			cpy.VotesPerStep[i] = vps.Copy()
		}
	}

	return *cpy
}

// NewStepVotesMsg creates a StepVotesMsg.
// Deprecated.
func NewStepVotesMsg(round uint64, hash []byte, sender []byte, sv StepVotes, step uint8) StepVotesMsg {
	return StepVotesMsg{
		Header: header.Header{
			Step:      step,
			Round:     round,
			BlockHash: hash,
			PubKeyBLS: sender,
		},
		StepVotes: sv,
	}
}

// Copy deeply the StepVotesMsg.
func (s StepVotesMsg) Copy() payload.Safe {
	b := new(bytes.Buffer)

	err := MarshalStepVotes(b, &s.StepVotes)
	if err != nil {
		log.WithError(err).Error("StepVotesMsg.Copy, could not MarshalStepVotes")
		// FIXME: creating a empty stepvotes with round 0 does not seem optimal, how can this be improved ?
		return NewStepVotesMsg(0, []byte{}, []byte{}, *NewStepVotes(), 0)
	}

	sv, err := UnmarshalStepVotes(b)
	if err != nil {
		// FIXME: creating a empty stepvotes with round 0 does not seem optimal, how can this be improved ?
		log.WithError(err).Error("StepVotesMsg.Copy, could not UnmarshalStepVotes")
		return NewStepVotesMsg(0, []byte{}, []byte{}, *NewStepVotes(), 0)
	}

	hdrCopy := s.Header.Copy()
	if hdrCopy == nil {
		return StepVotesMsg{
			// FIXME: creating a empty stepvotes with round 0 does not seem optimal, how can this be improved ?
			Header:    NewStepVotesMsg(0, []byte{}, []byte{}, *NewStepVotes(), 0).Header,
			StepVotes: *sv,
		}
	}

	cpy := StepVotesMsg{
		Header:    s.Header.Copy().(header.Header),
		StepVotes: *sv,
	}

	return cpy
}

// State returns the Header without information about Sender (as this is only
// for internal communications).
func (s StepVotesMsg) State() header.Header {
	return s.Header
}

// IsEmpty returns whether the StepVotesMsg represents a failed convergence
// attempt at consensus over a Reduction message.
func (s StepVotes) IsEmpty() bool {
	return s.Signature == nil
}

// String representation.
func (s StepVotes) String() string {
	var sb strings.Builder
	_, _ = sb.WriteString(fmt.Sprintf("BitSet: %d\n Sig: %v\n", s.BitSet, s.Signature))
	return sb.String()
}

// State returns the message header. This is to comply to the
// consensus.Message interface.
func (a Agreement) State() header.Header {
	return a.hdr
}

// Sender returns the BLS public key of the Sender.
func (a Agreement) Sender() []byte {
	return a.hdr.Sender()
}

// Cmp compares the big.Int representation of two agreement messages.
func (a Agreement) Cmp(other Agreement) int {
	return a.Repr.Cmp(other.Repr)
}

// SetSignature set a signature to the Agreement.
func (a *Agreement) SetSignature(signature []byte) {
	a.Repr = new(big.Int).SetBytes(signature)
	a.signature = signature
}

// Signature returns the signed vote.
func (a Agreement) Signature() []byte {
	return a.signature
}

// Equal checks if two agreement messages are the same.
func (a Agreement) Equal(aev Agreement) bool {
	return a.Repr.Cmp(aev.Repr) == 0
}

// GenerateCertificate is used by the Chain component.
func (a Agreement) GenerateCertificate() *block.Certificate {
	return &block.Certificate{
		StepOneBatchedSig: a.VotesPerStep[0].Signature,
		StepTwoBatchedSig: a.VotesPerStep[1].Signature,
		Step:              a.State().Step,
		StepOneCommittee:  a.VotesPerStep[0].BitSet,
		StepTwoCommittee:  a.VotesPerStep[1].BitSet,
	}
}

// UnmarshalAgreementMessage unmarshal a network inbound Agreement.
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

// NewStepVotes returns a new StepVotes structure for a given round, step and block hash.
func NewStepVotes() *StepVotes {
	return &StepVotes{
		BitSet:    uint64(0),
		Signature: nil,
	}
}

// Equal checks if two StepVotes structs are the same.
func (s *StepVotes) Equal(other *StepVotes) bool {
	return bytes.Equal(s.Signature, other.Signature)
}

// Add a vote to the StepVotes struct.
func (s *StepVotes) Add(signature []byte) error {
	if s.Signature == nil {
		s.Signature = signature
		return nil
	}

	var err error
	s.Signature, err = bls.AggregateSig(s.Signature, signature)
	return err
}

// MarshalAgreement marshals an Agreement event into a buffer.
func MarshalAgreement(r *bytes.Buffer, a Agreement) error {
	if err := header.Marshal(r, a.State()); err != nil {
		return err
	}

	// Marshal BLS Signature of VoteSet
	if err := encoding.WriteVarBytes(r, a.Signature()); err != nil {
		return err
	}

	// Marshal VotesPerStep
	if err := MarshalVotes(r, a.VotesPerStep); err != nil {
		return err
	}

	return nil
}

// UnmarshalAgreement unmarshals the buffer into an Agreement.
// Field order is the following:
// * Header [BLS Public Key; Round; Step]
// * Agreement [Signed Vote Set; Vote Set; BlockHash].
func UnmarshalAgreement(r *bytes.Buffer, a *Agreement) error {
	signature := make([]byte, 0)
	if err := encoding.ReadVarBytes(r, &signature); err != nil {
		log.WithError(err).Errorln("failed to unmarshal signature")
		return err
	}

	a.SetSignature(signature)

	votesPerStep := make([]*StepVotes, 2)
	if err := UnmarshalVotes(r, votesPerStep); err != nil {
		log.WithError(err).Errorln("failed to unmarshal step votes")
		return err
	}

	a.VotesPerStep = votesPerStep
	return nil
}

// NewAgreement returns an empty Agreement event. It is supposed to be used by
// the (secondstep reducer) for creating Agreement messages.
func NewAgreement(hdr header.Header) *Agreement {
	aggro := newAgreement()
	aggro.hdr = hdr
	return aggro
}

// newAgreement returns an empty Agreement event. It is used within the
// UnmarshalAgreement function.
// TODO: interface - []*StepVotes should not be references, but values.
func newAgreement() *Agreement {
	return &Agreement{
		hdr:          header.Header{},
		VotesPerStep: make([]*StepVotes, 2),
		signature:    make([]byte, 33),
		Repr:         new(big.Int),
	}
}

// SignAgreement signs an aggregated agreement event.
// XXX: either use this function or delete it!! Right now it is not used.
func SignAgreement(a *Agreement, keys key.Keys) error {
	buffer := new(bytes.Buffer)
	if err := MarshalVotes(buffer, a.VotesPerStep); err != nil {
		return err
	}

	signedVoteSet, err := bls.Sign(keys.BLSSecretKey, keys.BLSPubKey, buffer.Bytes())
	if err != nil {
		return err
	}

	a.SetSignature(signedVoteSet)
	return nil
}

// UnmarshalVotes unmarshals the array of StepVotes for a single Agreement.
func UnmarshalVotes(r *bytes.Buffer, votes []*StepVotes) error {
	length, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	// Agreement can only ever have two StepVotes, for the two
	// reduction steps.
	if length != 2 {
		return errors.New("malformed Agreement message: " + fmt.Sprintf("Got %d StepVotes (expected 2)", length))
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

// UnmarshalStepVotes unmarshals a single StepVote.
func UnmarshalStepVotes(r *bytes.Buffer) (*StepVotes, error) {
	sv := NewStepVotes()

	// BitSet
	var err error
	if err = encoding.ReadUint64LE(r, &sv.BitSet); err != nil {
		log.WithError(err).Errorln("failed to unmarshal stepvotes")
		return nil, err
	}

	// Signature
	signature := make([]byte, 0)
	if err = encoding.ReadVarBytes(r, &signature); err != nil {
		log.WithError(err).Errorln("failed to unmarshal signature")
		return nil, err
	}

	sv.Signature = signature
	return sv, nil
}

// MarshalVotes marshals an array of StepVotes.
func MarshalVotes(r *bytes.Buffer, votes []*StepVotes) error {
	if err := encoding.WriteVarInt(r, uint64(len(votes))); err != nil {
		log.WithError(err).Errorln("failed to marshal votes length")
		return err
	}

	if len(votes) != 2 {
		return errors.New("failed to marshal step votes, 2 votes are required")
	}

	for _, stepVotes := range votes {
		if err := MarshalStepVotes(r, stepVotes); err != nil {
			return err
		}
	}

	return nil
}

// MarshalStepVotes marshals the aggregated form of the BLS PublicKey and Signature
// for an ordered set of votes.
func MarshalStepVotes(r *bytes.Buffer, vote *StepVotes) error {
	if vote == nil || vote.Signature == nil {
		log.WithField("vote", vote).Error("could not MarshalStepVotes")
		return errors.New("invalid stepVotes")
	}

	// BitSet
	if err := encoding.WriteUint64LE(r, vote.BitSet); err != nil {
		return err
	}

	// Signature
	if err := encoding.WriteVarBytes(r, vote.Signature); err != nil {
		return err
	}

	return nil
}

// MockAgreement returns a mocked Agreement Event, to be used for testing purposes.
// It includes a vararg iterativeIdx to help avoiding duplicates when testing.
func MockAgreement(hash []byte, round uint64, step uint8, keys []key.Keys, p *user.Provisioners, iterativeIdx ...int) Agreement {
	// Make sure we create an event made by an actual voting committee member
	c := p.CreateVotingCommittee(mockSeed, round, step, len(keys))
	cKeys := createCommitteeKeySet(c, keys)

	idx := 0
	if len(iterativeIdx) != 0 {
		idx = iterativeIdx[0]
	}

	if idx > len(keys) {
		// FIXME: shall this panic ?
		panic("wrong iterative index: cannot iterate more than there are keys")
	}

	hdr := header.Header{Round: round, Step: step, BlockHash: hash, PubKeyBLS: cKeys[idx].BLSPubKey}
	a := NewAgreement(hdr)

	// generating reduction events (votes) and signing them
	steps := GenVotes(hash, mockSeed, round, step, keys, p)

	whole := new(bytes.Buffer)
	if err := header.MarshalSignableVote(whole, a.State()); err != nil {
		// FIXME: shall this panic ?
		panic(err)
	}

	sig, err := bls.Sign(cKeys[idx].BLSSecretKey, cKeys[idx].BLSPubKey, whole.Bytes())
	if err != nil {
		panic(err)
	}

	a.VotesPerStep = steps
	a.SetSignature(sig)
	return *a
}

// MockCommitteeVoteSet mocks a VoteSet.
func MockCommitteeVoteSet(p *user.Provisioners, k []key.Keys, hash []byte, committeeSize int, round uint64, step uint8) []Reduction {
	c1 := p.CreateVotingCommittee(mockSeed, round, step-2, len(k))
	c2 := p.CreateVotingCommittee(mockSeed, round, step-1, len(k))
	cKeys1 := createCommitteeKeySet(c1, k)
	cKeys2 := createCommitteeKeySet(c2, k)
	events := createVoteSet(cKeys1, cKeys2, hash, len(cKeys1), round, step)

	return events
}

// GenVotes randomly generates a slice of StepVotes with the indicated length.
// Albeit random, the generation is consistent with the rules of Votes.
func GenVotes(hash, seed []byte, round uint64, step uint8, keys []key.Keys, p *user.Provisioners) []*StepVotes {
	if len(keys) < 2 {
		// FIXME: shall this panic ?
		panic("At least two votes are required to mock an Agreement")
	}

	// Create committee key sets
	keySet1 := createCommitteeKeySet(p.CreateVotingCommittee(seed, round, step-1, len(keys)), keys)
	keySet2 := createCommitteeKeySet(p.CreateVotingCommittee(seed, round, step, len(keys)), keys)

	stepVotes1, set1 := createStepVotesAndSet(hash, round, step-1, keySet1)
	stepVotes2, set2 := createStepVotesAndSet(hash, round, step, keySet2)

	bitSet1 := createBitSet(set1, seed, round, step-1, len(keySet1), p)
	stepVotes1.BitSet = bitSet1
	bitSet2 := createBitSet(set2, seed, round, step, len(keySet2), p)
	stepVotes2.BitSet = bitSet2

	return []*StepVotes{stepVotes1, stepVotes2}
}

func createBitSet(set sortedset.Set, seed []byte, round uint64, step uint8, size int, p *user.Provisioners) uint64 {
	committee := p.CreateVotingCommittee(seed, round, step, size)
	return committee.Bits(set)
}

func createCommitteeKeySet(c user.VotingCommittee, k []key.Keys) (keys []key.Keys) {
	committeeKeys := c.MemberKeys()

	for _, cKey := range committeeKeys {
		for _, key := range k {
			if bytes.Equal(cKey, key.BLSPubKey) {
				keys = append(keys, key)
				break
			}
		}
	}

	return keys
}

func createStepVotesAndSet(hash []byte, round uint64, step uint8, keys []key.Keys) (*StepVotes, sortedset.Set) {
	set := sortedset.New()
	stepVotes := NewStepVotes()

	for _, k := range keys {
		// We should not aggregate any given key more than once.
		_, inserted := set.IndexOf(k.BLSPubKey)
		if !inserted {
			h := header.Header{
				BlockHash: hash,
				Round:     round,
				Step:      step,
				PubKeyBLS: k.BLSPubKey,
			}

			r := new(bytes.Buffer)
			if err := header.MarshalSignableVote(r, h); err != nil {
				// FIXME: shall this panic ?
				panic(err)
			}

			sigma, _ := bls.Sign(k.BLSSecretKey, k.BLSPubKey, r.Bytes())
			if err := stepVotes.Add(sigma); err != nil {
				// FIXME: shall this panic ?
				panic(err)
			}
		}

		set.Insert(k.BLSPubKey)
	}

	return stepVotes, set
}
