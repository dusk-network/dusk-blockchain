package sortition_test

import (
	"encoding/hex"
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/sortition"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestDeterministicSortition(t *testing.T) {
	// Set up a committee set with a stakes map
	stakes := make(map[string]uint64)
	var members [][]byte
	for i := 0; i < 1000; i++ {
		key, _ := crypto.RandEntropy(32)
		members = append(members, key)
		keyStr := hex.EncodeToString(key)
		stakes[keyStr] = 500
	}

	// Run sortition to get 50 members out
	committee, err := sortition.CreateCommittee(100, 50000, 1, members, stakes)
	if err != nil {
		t.Fatal(err)
	}

	// Committee should be 50 members
	if len(committee) != 50 {
		t.Fatal("committee size is not correct")
	}

	// All members should pass verification with at least one vote
	for _, member := range committee {
		votes := sortition.Verify(committee, member)
		if votes == 0 {
			t.Fatal("returned committee contained wrong member")
		}
	}
}
