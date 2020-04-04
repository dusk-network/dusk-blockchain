package user

import (
	"bytes"
	"errors"
	"math/rand"
)

// Bid is the 32 byte X value, created from a bidding transaction amount and M.
type Bid struct {
	X         [32]byte
	M         [32]byte
	EndHeight uint64
}

// Equals will return whether or not the two bids are the same.
func (b Bid) Equals(bid Bid) bool {
	return bytes.Equal(b.X[:], bid.X[:])
}

// BidList is a list of bid X values.
type BidList []Bid

// ReconstructBidListSubset will turn a slice of bytes into a BidList.
func ReconstructBidListSubset(pl []byte) (BidList, error) {
	if len(pl)%32 != 0 {
		return nil, errors.New("malformed bidlist")
	}

	numBids := len(pl) / 32
	r := bytes.NewReader(pl)
	bl := make(BidList, numBids)
	for i := 0; i < numBids; i++ {
		var bid Bid
		if _, err := r.Read(bid.X[:]); err != nil {
			return nil, err
		}

		bl[i] = bid
	}

	return bl, nil
}

// ValidateBids will check if the passed BidList subset contains valid bids.
func (b BidList) ValidateBids(bidListSubset BidList) error {
loop:
	for _, x := range bidListSubset {
		for _, x2 := range b {
			if x.Equals(x2) {
				continue loop
			}
		}

		return errors.New("invalid public list")
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

// Remove idx from BidList
func (b *BidList) Remove(idx int) {
	list := *b
	if idx == len(list)-1 {
		list = list[:idx]
	} else {
		list = append(list[:idx], list[idx+1:]...)
	}
	*b = list
}
