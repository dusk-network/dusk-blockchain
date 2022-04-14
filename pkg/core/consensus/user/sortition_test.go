// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package user_test

import (
	"bytes"
	"math/big"
	"sort"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
	"github.com/stretchr/testify/assert"
)

// Test if removing members from the VotingCommittee works properly.
func TestRemove(t *testing.T) {
	committee := &user.VotingCommittee{sortedset.NewCluster()}

	nr := 5
	for i := 0; i < nr; i++ {
		k := key.NewRandKeys()
		bk := (&big.Int{}).SetBytes(k.BLSPubKey)
		committee.Set = append(committee.Set, bk)
	}

	sort.Sort(committee.Set)

	lastElem := committee.Set[nr-1].Bytes()
	committee.Set.Remove(lastElem)

	i, found := committee.Set.IndexOf(lastElem)
	assert.False(t, found)

	assert.Equal(t, nr-1, i)
}

// Test if MemberKeys returns all public keys in the correct order.
func TestMemberKeys(t *testing.T) {
	p, ks := consensus.MockProvisioners(50)

	sks := sortedKeys{}
	sks = append(sks, ks...)

	sort.Sort(sks)

	seed := []byte{0, 0, 0, 0}
	v := p.CreateVotingCommittee(seed, 1, 1, 50)

	mk := v.MemberKeys()
	assert.Equal(t, 50, len(mk))

	for i := 0; i < 3; i++ {
		assert.True(t, bytes.Equal(mk[i], v.Set[i].Bytes()))
	}
}

// Test that stakes are reduced properly during sortition, and that the changes
// do not persist outside of sortition.
func TestSubtractStake(t *testing.T) {
	p, ks := consensus.MockProvisioners(50)
	// The mocked provisioner set will attribute only 500 atomic units of DUSK
	// to each provisioner, which means they can only be extracted once per step.
	// Therefore, we will check that a call to `CreateVotingCommittee` will
	// extract every single member in the set, with no duplicates.
	seed := []byte{0, 0, 0, 0}

	v := p.CreateVotingCommittee(seed, 1, 1, 50)
	duplicates := make(map[string]struct{})

	for _, k := range v.Set {
		if _, ok := duplicates[string(k.Bytes())]; ok {
			t.Fatal("duplicate member was found in the voting committee")
		}

		duplicates[string(k.Bytes())] = struct{}{}
	}

	// Check that `CreateVotingCommittee` did not modify our original provisioner set
	for _, k := range ks {
		stake, err := p.GetStake(k.BLSPubKey)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, uint64(500), stake)
	}
}

// Test that removing a stake from the array can not cause a panic.
func TestRemoveStake(t *testing.T) {
	p, _ := consensus.MockProvisioners(10)

	// Let's add two stakes which are not yet active to one of the provisioners.
	m, err := p.MemberAt(2)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	m.AddStake(user.Stake{500, 0, 1000, 10000})

	m, err = p.MemberAt(2)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	m.AddStake(user.Stake{500, 0, 1000, 10000})

	// Now, extract a committee for round 1 step 1
	seed := []byte{0, 0, 0, 0}

	assert.NotPanics(t, func() { p.CreateVotingCommittee(seed, 1, 1, 10) })
}
