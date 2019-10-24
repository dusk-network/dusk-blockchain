package agreement

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-crypto/bls"
)

var _ wire.Event = (*Agreement)(nil)

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

	// Agreement is the Event created at the end of the Reduction process. It includes
	// the aggregated compressed signatures of all voters
	Agreement struct {
		header.Header
		signedVotes  []byte
		VotesPerStep []*StepVotes
		intRepr      *big.Int
	}
)

func (a Agreement) Cmp(other Agreement) int {
	return a.intRepr.Cmp(other.intRepr)
}

func (a *Agreement) SetSignature(signedVotes []byte) {
	a.intRepr = new(big.Int).SetBytes(signedVotes)
	a.signedVotes = signedVotes
}

func (a Agreement) SignedVotes() []byte {
	return a.signedVotes
}

func (a Agreement) Sender() []byte {
	return a.Header.Sender()
}

func (a Agreement) Equal(ev wire.Event) bool {
	aev, ok := ev.(Agreement)
	if !ok {
		return false
	}

	return a.Header.Equal(aev.Header) && a.intRepr.Cmp(aev.intRepr) == 0
}

// NewStepVotes returns a new StepVotes structure for a given round, step and block hash
func NewStepVotes() *StepVotes {
	return &StepVotes{
		Apk:       &bls.Apk{},
		BitSet:    uint64(0),
		Signature: &bls.Signature{},
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
	if sv.Step == uint8(0) {
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

// Marshal an Agreement event into a buffer.
func Marshal(r *bytes.Buffer, a Agreement) error {
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
func Unmarshal(r *bytes.Buffer, a *Agreement) error {
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

// New returns an empty Agreement event.
func New(h header.Header) *Agreement {
	return &Agreement{
		VotesPerStep: make([]*StepVotes, 2),
		signedVotes:  make([]byte, 33),
		intRepr:      new(big.Int),
		Header:       h,
	}
}

// Sign signs an aggregated agreement event
func Sign(a *Agreement, keys user.Keys) error {
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

