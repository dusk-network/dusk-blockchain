package user

import (
	"bytes"
	"errors"
	"math/rand"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"
)

// Bid is the 32 byte X value, created from a bidding transaction amount and M.
type Bid [32]byte

// Equals will return whether or not the two bids are the same.
func (b Bid) Equals(bid Bid) bool {
	return bytes.Equal(b[:], bid[:])
}

// BidList is a list of bid X values.
type BidList []Bid

// ReconstructBidListSubset will turn a slice of bytes into a BidList.
func ReconstructBidListSubset(pl []byte) (BidList, *prerror.PrError) {
	if len(pl)%32 != 0 {
		return nil, prerror.New(prerror.Low, errors.New("malformed bidlist"))
	}

	numBids := len(pl) / 32
	r := bytes.NewReader(pl)
	BidList := make(BidList, numBids)
	for i := 0; i < numBids; i++ {
		var bid Bid
		if _, err := r.Read(bid[:]); err != nil {
			return nil, prerror.New(prerror.High, err)
		}

		BidList[i] = bid
	}

	return BidList, nil
}

// ValidateBids will check if the passed BidList subset contains valid bids.
func (b BidList) ValidateBids(bidListSubset BidList) *prerror.PrError {
loop:
	for _, x := range bidListSubset {
		for _, x2 := range b {
			if x.Equals(x2) {
				continue loop
			}
		}

		return prerror.New(prerror.Low, errors.New("invalid public list"))
	}

	return nil
}

// Subset will shuffle the BidList, and returns a specified amount of
// bids from it.
func (b BidList) Subset(amount int) []Bid {
	// Shuffle the public list
	rand.Shuffle(len(b), func(i, j int) { b[i], b[j] = b[j], b[i] })

	// Create our subset
	subset := make([]Bid, amount)
	for i := 0; i < amount; i++ {
		subset[i] = b[i]
	}

	return subset
}

// Contains checks if the BidList contains a specified Bid.
func (b BidList) Contains(bid Bid) bool {
	for _, x := range b {
		if x.Equals(bid) {
			return true
		}
	}

	return false
}

// AddBid will add a bid to the BidList.
func (b *BidList) AddBid(bid Bid) {
	// Check for duplicates
	for _, bidFromList := range *b {
		if bidFromList.Equals(bid) {
			return
		}
	}

	*b = append(*b, bid)
}

// RemoveBid will iterate over a BidList and remove a specified bid.
func (b *BidList) RemoveBid(bid Bid) {
	for i, bidFromList := range *b {
		if bidFromList.Equals(bid) {
			list := *b
			list = append(list[:i], list[i+1:]...)
			*b = list
		}
	}
}
