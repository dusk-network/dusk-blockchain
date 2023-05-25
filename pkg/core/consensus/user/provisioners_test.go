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

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/stretchr/testify/assert"
)

// Test that creation of a voting committee from a set of Provisioners works as intended.
func TestCreateVotingCommittee(t *testing.T) {
	// Set up a committee set with a stakes map
	p, _ := consensus.MockProvisioners(50)

	// Run sortition to get 50 members
	seed := []byte{0, 0, 0, 0}
	committee := p.CreateVotingCommittee(seed, 100, 1, 50)

	// total amount of members in the committee should be 50
	assert.Equal(t, 50, committee.Size())
}

type sortedKeys []key.Keys

func (s sortedKeys) Len() int      { return len(s) }
func (s sortedKeys) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s sortedKeys) Less(i, j int) bool {
	return btoi(s[i]).Cmp(btoi(s[j])) < 0
}

func btoi(k key.Keys) *big.Int {
	b := k.BLSPubKey
	return (&big.Int{}).SetBytes(b)
}

// Test that provisioners are sorted properly.
func TestMemberAt(t *testing.T) {
	var ks sortedKeys

	nr := 50

	p, k := consensus.MockProvisioners(nr)
	ks = append(ks, k...)

	sort.Sort(ks)

	for i := 0; i < nr; i++ {
		m, err := p.MemberAt(i)
		if !assert.NoError(t, err) {
			t.FailNow()
		}

		assert.True(t, bytes.Equal(m.PublicKeyBLS, ks[i].BLSPubKey))
	}
}

// Add a member, and check if the Get functions are working properly.
func TestGetMember(t *testing.T) {
	// Set up a committee set with a stakes map
	tKeys := make([][]byte, 0)

	p, k := consensus.MockProvisioners(50)
	for _, keys := range k {
		tKeys = append(tKeys, keys.BLSPubKey)
	}

	_, err := p.GetStake([]byte("Fake Public Key"))
	assert.Error(t, err)

	for _, tk := range tKeys {
		m := p.GetMember(tk)
		s, _ := p.GetStake(tk)

		assert.Equal(t, 1000*config.DUSK, s)
		assert.Equal(t, m.PublicKeyBLS, tk)
	}
}

func TestMarshalProvisioners(t *testing.T) {
	p, _ := consensus.MockProvisioners(1)

	buf := bytes.Buffer{}

	err := user.MarshalProvisioners(&buf, p)
	assert.NoError(t, err)

	p2, err := user.UnmarshalProvisioners(&buf)
	assert.NoError(t, err)

	assert.True(t, p.Set.Equal(p2.Set))
}
