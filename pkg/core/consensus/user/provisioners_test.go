package user_test

import (
	"bytes"
	"math/big"
	"sort"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/stretchr/testify/assert"
)

// Test that creation of a voting committee from a set of Provisioners works as intended.
func TestCreateVotingCommittee(t *testing.T) {
	// Set up a committee set with a stakes map
	p := &user.Provisioners{}
	var totalWeight uint64
	for i := 0; i < 50; i++ {
		keys, _ := user.NewRandKeys()
		member := &user.Member{}
		member.PublicKeyEd = keys.EdPubKeyBytes
		member.PublicKeyBLS = keys.BLSPubKeyBytes
		member.Stakes = make([]user.Stake, 1)
		member.Stakes[0].Amount = 500
		member.Stakes[0].EndHeight = 10000
		p.Members[string(keys.BLSPubKeyBytes)] = member

		totalWeight += 500
	}

	// Run sortition to get 50 members (as a Set, committee cannot contain any duplicate)
	committee := p.CreateVotingCommittee(100, totalWeight, 1, 50)

	// total amount of members in the committee should be 50
	assert.Equal(t, 50, committee.Size())
}

type sortedKeys []user.Keys

func (s sortedKeys) Len() int      { return len(s) }
func (s sortedKeys) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s sortedKeys) Less(i, j int) bool {

	return btoi(s[i]).Cmp(btoi(s[j])) < 0
}

func btoi(k user.Keys) *big.Int {
	b := k.BLSPubKeyBytes
	return (&big.Int{}).SetBytes(b)
}

// Test that provisioners are sorted properly.
func TestMemberAt(t *testing.T) {
	nr := 50
	p := &user.Provisioners{}
	var ks sortedKeys
	for i := 0; i < nr; i++ {
		keys, _ := user.NewRandKeys()
		member := &user.Member{}
		member.PublicKeyEd = keys.EdPubKeyBytes
		member.PublicKeyBLS = keys.BLSPubKeyBytes
		member.Stakes = make([]user.Stake, 1)
		member.Stakes[0].Amount = 500
		member.Stakes[0].EndHeight = 10000
		p.Members[string(keys.BLSPubKeyBytes)] = member
		ks = append(ks, keys)
	}

	sort.Sort(ks)

	for i := 0; i < nr; i++ {
		m := p.MemberAt(i)
		assert.True(t, bytes.Equal(m.PublicKeyBLS, ks[i].BLSPubKeyBytes))
	}
}

// Add a member, and check if the Get functions are working properly.
func TestGetMember(t *testing.T) {
	// Set up a committee set with a stakes map
	tKeys := make([][]byte, 0)
	p := &user.Provisioners{}
	for i := 0; i < 50; i++ {
		keys, _ := user.NewRandKeys()
		member := &user.Member{}
		member.PublicKeyEd = keys.EdPubKeyBytes
		member.PublicKeyBLS = keys.BLSPubKeyBytes
		member.Stakes = make([]user.Stake, 1)
		member.Stakes[0].Amount = 500
		member.Stakes[0].EndHeight = 10000
		p.Members[string(keys.BLSPubKeyBytes)] = member
		tKeys = append(tKeys, keys.BLSPubKeyBytes)
	}

	_, err := p.GetStake([]byte("Fake Public Key"))
	assert.Error(t, err)
	for _, tk := range tKeys {
		m := p.GetMember(tk)
		s, _ := p.GetStake(tk)
		assert.Equal(t, uint64(500), s)
		assert.Equal(t, m.PublicKeyBLS, tk)
	}
}
