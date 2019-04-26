package user_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
)

func TestCreateVotingCommittee(t *testing.T) {
	// Set up a committee set with a stakes map
	p := user.Provisioners{}
	var totalWeight uint64
	for i := 0; i < 50; i++ {
		keys, _ := user.NewRandKeys()
		if err := p.AddMember(keys.EdPubKeyBytes(), keys.BLSPubKey.Marshal(), 500); err != nil {
			t.Fatal(err)
		}

		totalWeight += 500
	}

	// Run sortition to get 50 members (as a Set, committee cannot contain any duplicate)
	committee := p.CreateVotingCommittee(100, totalWeight, 1)

	// total amount of members in the committee should be 50
	assert.Equal(t, 50, len(committee))
}

func TestIndexOf(t *testing.T) {
	// Set up a committee set with a stakes map
	tKeys := make([][]byte, 0)
	p := user.Provisioners{}
	for i := 0; i < 50; i++ {
		keys, _ := user.NewRandKeys()
		if err := p.AddMember(keys.EdPubKeyBytes(), keys.BLSPubKey.Marshal(), 500); err != nil {
			t.Fatal(err)
		}

		if rand.Intn(100) < 30 {
			tKeys = append(tKeys, keys.BLSPubKey.Marshal())
		}
	}

	for _, tk := range tKeys {
		i, found := p.IndexOf(tk)
		assert.True(t, found)
		assert.Equal(t, p[i].PublicKeyBLS.Marshal(), tk)
	}

}
