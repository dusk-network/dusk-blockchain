package user_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestAddAndRemoveBid(t *testing.T) {
	bidList := &user.BidList{}
	bid := createBid()
	bidList.AddBid(bid)
	assert.True(t, bidList.Contains(bid))

	bidList.RemoveBid(bid)
	assert.False(t, bidList.Contains(bid))
}

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

func TestRemoveExpired(t *testing.T) {
	bidList := createBidList(10)

	// All bids have their end height at 1000 - so let's remove them all
	bidList.RemoveExpired(1001)

	assert.Equal(t, 0, len(*bidList))
}

func createBidList(amount int) *user.BidList {
	bidList := &user.BidList{}
	for i := 0; i < amount; i++ {
		bid := createBid()
		bidList.AddBid(bid)
	}

	return bidList
}

func createBid() user.Bid {
	var bid user.Bid
	bidSlice, _ := crypto.RandEntropy(32)
	copy(bid.X[:], bidSlice)
	bid.EndHeight = 1000
	return bid
}
