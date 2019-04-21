package user_test

import (
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
	assert.Equal(t, 50, committee.Size())
}
