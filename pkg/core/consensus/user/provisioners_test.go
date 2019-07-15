package user_test

import (
	"bytes"
	"math/big"
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/lite"
)

// Test that creation of a voting committee from a set of Provisioners works as intended.
func TestCreateVotingCommittee(t *testing.T) {
	// Set up a committee set with a stakes map
	_, db := lite.CreateDBConnection()
	p, _, err := user.NewProvisioners(db)
	if err != nil {
		t.Fatal(err)
	}
	var totalWeight uint64
	for i := 0; i < 50; i++ {
		keys, _ := user.NewRandKeys()
		if err := p.AddMember(keys.EdPubKeyBytes, keys.BLSPubKeyBytes, 500, 0, 1000); err != nil {
			t.Fatal(err)
		}

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
	_, db := lite.CreateDBConnection()
	p, _, err := user.NewProvisioners(db)
	if err != nil {
		t.Fatal(err)
	}
	var ks sortedKeys
	for i := 0; i < nr; i++ {
		keys, _ := user.NewRandKeys()
		if err := p.AddMember(keys.EdPubKeyBytes, keys.BLSPubKeyBytes, 500, 0, 1000); err != nil {
			t.Fatal(err)
		}
		ks = append(ks, keys)
	}

	sort.Sort(ks)

	for i := 0; i < nr; i++ {
		m := p.MemberAt(i)
		assert.True(t, bytes.Equal(m.PublicKeyBLS.Marshal(), ks[i].BLSPubKeyBytes))
	}
}

// Add a member, and check if the Get functions are working properly.
func TestAddGetMember(t *testing.T) {
	// Set up a committee set with a stakes map
	tKeys := make([][]byte, 0)
	_, db := lite.CreateDBConnection()
	p, _, err := user.NewProvisioners(db)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 50; i++ {
		keys, _ := user.NewRandKeys()
		if err := p.AddMember(keys.EdPubKeyBytes, keys.BLSPubKeyBytes, 500, 0, 1000); err != nil {
			t.Fatal(err)
		}

		if rand.Intn(100) < 30 {
			tKeys = append(tKeys, keys.BLSPubKeyBytes)
		}
	}
	_, err = p.GetStake([]byte("Fake Public Key"))
	assert.Error(t, err)
	for _, tk := range tKeys {
		m := p.GetMember(tk)
		s, _ := p.GetStake(tk)
		assert.Equal(t, uint64(500), s)
		assert.Equal(t, m.PublicKeyBLS.Marshal(), tk)
	}
}

// Add and then a remove a provisioner, to check if removal works properly.
func TestRemove(t *testing.T) {
	_, db := lite.CreateDBConnection()
	p, _, err := user.NewProvisioners(db)
	if err != nil {
		t.Fatal(err)
	}
	keys, _ := user.NewRandKeys()
	if err := p.AddMember(keys.EdPubKeyBytes, keys.BLSPubKeyBytes, 500, 0, 1000); err != nil {
		t.Fatal(err)
	}

	if !p.Remove(keys.BLSPubKeyBytes) {
		t.Fatal("could not remove a member we just added")
	}

	assert.Zero(t, p.Size(1))
}
