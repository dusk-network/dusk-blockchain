package user_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

func TestSubset(t *testing.T) {
	bidList := createBidList(10)
	subset := bidList.Subset(5)
	assert.Subset(t, *bidList, subset)
}

func TestValidateBids(t *testing.T) {
	bidList := createBidList(10)
	subset := bidList.Subset(5)

	// use assert.Nil here instead of assert.NoError, due to the use of PrError
	assert.Nil(t, bidList.ValidateBids(subset))

	// add a bid to the subset, unknown to the bidlist
	bid := createBid()
	subset = append(subset, bid)
	assert.NotNil(t, bidList.ValidateBids(subset))
}

func TestReconstructBidListSubset(t *testing.T) {
	bidList := createBidList(10)

	// Turn it into a straight slice of bytes
	bidListBytes := make([]byte, 0)
	for _, bid := range *bidList {
		bidListBytes = append(bidListBytes, bid.X[:]...)
	}

	// Now reconstruct it, and check if it is equal to the initial bidList
	reconstructed, err := user.ReconstructBidListSubset(bidListBytes)
	if err != nil {
		t.Fatal(err)
	}

	for i := range *bidList {
		assert.True(t, bidList.Contains(reconstructed[i]))
	}
}

func createBidList(amount int) *user.BidList {
	bidlist := &user.BidList{}
	for i := 0; i < amount; i++ {
		bid := createBid()
		*bidlist = append(*bidlist, bid)
	}

	return bidlist
}

func createBid() user.Bid {
	var bid user.Bid
	bidSlice, _ := crypto.RandEntropy(32)
	copy(bid.X[:], bidSlice)
	bid.EndHeight = 1000
	return bid
}
