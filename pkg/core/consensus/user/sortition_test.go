package user_test

import (
	"bytes"
	"math/big"
	"sort"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
	"github.com/stretchr/testify/assert"
)

// Test if removing members from the VotingCommittee works properly.
func TestRemove(t *testing.T) {
	nr := 5
	committee := &user.VotingCommittee{sortedset.New()}
	for i := 0; i < nr; i++ {
		k, _ := user.NewRandKeys()
		bk := (&big.Int{}).SetBytes(k.BLSPubKeyBytes)
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
	v := p.CreateVotingCommittee(1, 1, 50)
	mk := v.MemberKeys()
	assert.Equal(t, 50, len(mk))
	for i := 0; i < 3; i++ {
		assert.True(t, bytes.Equal(mk[i], v.Set[i].Bytes()))
	}

}
