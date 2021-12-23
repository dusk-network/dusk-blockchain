// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package committee

import (
	"math"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
)

// PregenerationAmount is the size of a pregenerated committee.
var PregenerationAmount uint8 = 1

// Handler is injected in the consensus components that work with the various
// committee. It generates and maintains a list of active and valid committee members and
// handle the votes.
type Handler struct {
	key.Keys
	Provisioners user.Provisioners
	Committees   []user.VotingCommittee
	seed         []byte

	lock sync.RWMutex
}

// NewHandler creates a new committee.Handler by instantiating the committee
// slice, setting the keys and setting the Provisioner set.
func NewHandler(keys key.Keys, p user.Provisioners, seed []byte) *Handler {
	return &Handler{
		Keys:         keys,
		Committees:   make([]user.VotingCommittee, math.MaxUint8),
		Provisioners: p,
		seed:         seed,
	}
}

// AmMember checks if we are part of the committee for a given round and step.
func (b *Handler) AmMember(round uint64, step uint8, maxSize int) bool {
	return b.IsMember(b.Keys.BLSPubKey, round, step, maxSize)
}

// IsMember checks if a provisioner with a given BLS public key is
// part of the committee for a given round and step.
func (b *Handler) IsMember(pubKeyBLS []byte, round uint64, step uint8, maxSize int) bool {
	return b.Committee(round, step, maxSize).IsMember(pubKeyBLS)
}

// VotesFor returns the amount of votes for a public key for a given round and step.
func (b *Handler) VotesFor(pubKeyBLS []byte, round uint64, step uint8, maxSize int) int {
	return b.Committee(round, step, maxSize).OccurrencesOf(pubKeyBLS)
}

// Committee returns a VotingCommittee for a given round and step.
func (b *Handler) Committee(round uint64, step uint8, maxSize int) user.VotingCommittee {
	if b.membersAt(step) == 0 {
		b.generateCommittees(b.seed, round, step, maxSize)
	}

	b.lock.RLock()
	committee := b.Committees[step]
	b.lock.RUnlock()

	return committee
}

func (b *Handler) generateCommittees(seed []byte, round uint64, step uint8, maxSize int) {
	size := b.CommitteeSize(round, maxSize)

	b.lock.Lock()
	defer b.lock.Unlock()

	committees := b.Provisioners.GenerateCommittees(seed, round, PregenerationAmount, step, size)
	for i, committee := range committees {
		if step == math.MaxUint8 {
			panic("Consensus reached max steps")
		}

		b.Committees[int(step)+i] = committee
	}
}

// CommitteeSize returns the size of a VotingCommittee, depending on
// how many provisioners are in the set.
func (b *Handler) CommitteeSize(round uint64, maxSize int) int {
	b.lock.RLock()
	size := b.Provisioners.SubsetSizeAt(round)
	b.lock.RUnlock()

	if size > maxSize {
		return maxSize
	}

	return size
}

func (b *Handler) membersAt(idx uint8) int {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if idx == math.MaxUint8 {
		panic("Consensus reached max steps")
	}

	return b.Committees[idx].Set.Len()
}
