package user_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestDeterministicSortition(t *testing.T) {
	// Set up a committee set with a stakes map
	stakes := make(map[string]uint64)
	c := user.Committee{}
	for i := 0; i < 1000; i++ {
		key, _ := crypto.RandEntropy(32)
		if err := c.AddMember(key); err != nil {
			t.Fatal(err)
		}

		keyStr := hex.EncodeToString(key)
		stakes[keyStr] = 500
	}

	info := &user.ConsensusInfo{
		TotalWeight:   500000,
		Committee:     c,
		NodeStakes:    stakes,
		CommitteeSize: 50,
		VoteLimit:     38,
	}

	// Run sortition to get 50 members out
	committee, err := info.CreateVotingCommittee(100, 1)
	if err != nil {
		t.Fatal(err)
	}

	// All members should pass verification with at least one vote
	var total uint8
	for pk, _ := range committee {
		if committee[pk] == 0 {
			t.Fatal("returned committee contained wrong member")
		}

		total += committee[pk]
	}

	assert.Equal(t, uint8(50), total)
}
