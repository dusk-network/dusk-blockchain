// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package message

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dusk-network/bls12_381-sign/bls"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
)

type (
	// Reduction is one of the messages used in the consensus algorithms. As
	// such it encapsulates a header.Header to allow the Coordinator to
	// correctly enforce the sequence of state changes expressed through the algorithm.
	Reduction struct {
		hdr        header.Header
		SignedHash []byte
	}
)

// NewReduction returns and empty Reduction event.
func NewReduction(hdr header.Header) *Reduction {
	return &Reduction{
		hdr:        hdr,
		SignedHash: make([]byte, 0),
	}
}

// Copy complies with the Safe interface.
func (r Reduction) Copy() payload.Safe {
	cpy := Reduction{}

	cpy.hdr = r.hdr.Copy().(header.Header)
	if r.SignedHash != nil {
		cpy.SignedHash = make([]byte, len(r.SignedHash))
		copy(cpy.SignedHash, r.SignedHash)
	}

	return cpy
}

func (r Reduction) String() string {
	var sb strings.Builder

	_, _ = sb.WriteString(r.hdr.String())
	_, _ = sb.WriteString(" signature' ")
	_, _ = sb.WriteString(util.StringifyBytes(r.SignedHash))
	_, _ = sb.WriteString("'")

	return sb.String()
}

// State is used to comply to consensus.Message.
func (r Reduction) State() header.Header {
	return r.hdr
}

// Sender is used to comply to consensus.Message.
func (r Reduction) Sender() []byte {
	return r.hdr.Sender()
}

// Equal is used to comply to consensus.Message.
func (r Reduction) Equal(msg Message) bool {
	m, ok := msg.Payload().(Reduction)
	return ok && r.hdr.Equal(m.hdr) && bytes.Equal(r.SignedHash, m.SignedHash)
}

// MarshalJSON ...
func (r Reduction) MarshalJSON() ([]byte, error) {
	v := fmt.Sprintf("Signature: %s, Hash: %s, Round: %d, Step: %d, Sender key %s)", util.StringifyBytes(r.SignedHash),
		util.StringifyBytes(r.hdr.BlockHash), r.hdr.Round, r.hdr.Step, util.StringifyBytes(r.hdr.PubKeyBLS))

	return json.Marshal(v)
}

// UnmarshalReductionMessage unmarshals a serialization from a buffer.
func UnmarshalReductionMessage(r *bytes.Buffer, m SerializableMessage) error {
	bev := NewReduction(header.Header{})
	if err := UnmarshalReduction(r, bev); err != nil {
		return err
	}

	m.SetPayload(*bev)
	return nil
}

// UnmarshalReduction unmarshals the buffer into a Reduction event.
func UnmarshalReduction(r *bytes.Buffer, bev *Reduction) error {
	if err := header.Unmarshal(r, &bev.hdr); err != nil {
		return err
	}

	bev.SignedHash = make([]byte, 0)
	return encoding.ReadVarBytes(r, &bev.SignedHash)
}

// MarshalReduction a Reduction event into a buffer.
func MarshalReduction(r *bytes.Buffer, bev Reduction) error {
	if err := header.Marshal(r, bev.State()); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, bev.SignedHash); err != nil {
		return err
	}

	return nil
}

// UnmarshalVoteSet unmarshals a Reduction slice from a buffer.
func UnmarshalVoteSet(r *bytes.Buffer) ([]Reduction, error) {
	length, err := encoding.ReadVarInt(r)
	if err != nil {
		return nil, err
	}

	evs := make([]Reduction, length)

	for i := uint64(0); i < length; i++ {
		rev := NewReduction(header.Header{})
		if err := UnmarshalReduction(r, rev); err != nil {
			return nil, err
		}

		evs[i] = *rev
	}

	return evs, nil
}

// MarshalVoteSet marshals a slice of Reduction events to a buffer.
func MarshalVoteSet(r *bytes.Buffer, evs []Reduction) error {
	if err := encoding.WriteVarInt(r, uint64(len(evs))); err != nil {
		return err
	}

	for _, event := range evs {
		if err := MarshalReduction(r, event); err != nil {
			return err
		}
	}

	return nil
}

/********************/
/* MOCKUP FUNCTIONS */
/********************/

// MockReduction mocks a Reduction event and returns it.
// It includes a vararg iterativeIdx to help avoiding duplicates when testing.
func MockReduction(hash []byte, round uint64, step uint8, keys []key.Keys, iterativeIdx ...int) Reduction {
	idx := 0
	if len(iterativeIdx) != 0 {
		idx = iterativeIdx[0]
	}

	if idx > len(keys) {
		panic("wrong iterative index: cannot iterate more than there are keys")
	}

	hdr := header.Header{Round: round, Step: step, BlockHash: hash, PubKeyBLS: keys[idx].BLSPubKey}
	r := new(bytes.Buffer)
	_ = header.MarshalSignableVote(r, hdr)

	sigma, err := bls.Sign(keys[idx].BLSSecretKey, keys[idx].BLSPubKey, r.Bytes())
	if err != nil {
		panic(err)
	}

	return Reduction{
		hdr:        hdr,
		SignedHash: sigma,
	}
}

// MockVotes mocks a slice of Reduction events and returns it.
func MockVotes(hash []byte, round uint64, step uint8, keys []key.Keys, amount int) []Reduction {
	var voteSet []Reduction

	for i := 0; i < amount; i++ {
		r := MockReduction(hash, round, step, keys, i)
		voteSet = append(voteSet, r)
	}

	return voteSet
}

// MockVoteSet mocks a slice of Reduction events for two adjacent steps,
// and returns it.
func MockVoteSet(hash []byte, round uint64, step uint8, keys []key.Keys, amount int) []Reduction {
	if step < uint8(2) {
		panic("Need at least 2 steps to create an Agreement")
	}

	votes1 := MockVotes(hash, round, step-1, keys, amount)
	votes2 := MockVotes(hash, round, step, keys, amount)
	return append(votes1, votes2...)
}

// createVoteSet creates and returns a set of Reduction votes for two steps.
func createVoteSet(k1, k2 []key.Keys, hash []byte, size int, round uint64, step uint8) (events []Reduction) {
	// We can not have duplicates in the vote set.
	duplicates := make(map[string]struct{})
	// We need 75% of the committee size worth of events to reach quorum.
	for j := 0; j < int(float64(size)*0.67); j++ {
		if _, ok := duplicates[string(k1[j].BLSPubKey)]; !ok {
			ev := MockReduction(hash, round, step-2, k1, j)
			events = append(events, ev)
			duplicates[string(k1[j].BLSPubKey)] = struct{}{}
		}
	}

	// Clear the duplicates map, since we will most likely have identical keys in each array
	for k := range duplicates {
		delete(duplicates, k)
	}

	for j := 0; j < int(float64(size)*0.67); j++ {
		if _, ok := duplicates[string(k2[j].BLSPubKey)]; !ok {
			ev := MockReduction(hash, round, step-1, k2, j)
			events = append(events, ev)
			duplicates[string(k2[j].BLSPubKey)] = struct{}{}
		}
	}

	return events
}
