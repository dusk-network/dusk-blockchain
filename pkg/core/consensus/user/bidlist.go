package user

import (
	"bytes"
	"errors"
	"math/rand"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
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
	BidList := make(BidList, numBids)
	for i := 0; i < numBids; i++ {
		var bid Bid
		if _, err := r.Read(bid.X[:]); err != nil {
			return nil, err
		}

		BidList[i] = bid
	}

	return BidList, nil
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

func (b *BidList) Remove(idx int) {
	list := *b
	if idx == len(list)-1 {
		list = list[:idx]
	} else {
		list = append(list[:idx], list[idx+1:]...)
	}
	*b = list
}

func MarshalBidList(r *bytes.Buffer, bidList BidList) error {
	if err := encoding.WriteVarInt(r, uint64(len(bidList))); err != nil {
		return err
	}

	for _, bid := range bidList {
		if err := marshalBid(r, bid); err != nil {
			return err
		}
	}

	return nil
}

func marshalBid(r *bytes.Buffer, bid Bid) error {
	if err := encoding.Write256(r, bid.X[:]); err != nil {
		return err
	}

	if err := encoding.Write256(r, bid.M[:]); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, bid.EndHeight); err != nil {
		return err
	}

	return nil
}

func UnmarshalBidList(r *bytes.Buffer) (BidList, error) {
	lBidList, err := encoding.ReadVarInt(r)
	if err != nil {
		return BidList{}, err
	}

	bidList := make([]Bid, lBidList)
	for i := uint64(0); i < lBidList; i++ {
		if err := unmarshalBid(r, &bidList[i]); err != nil {
			return BidList{}, err
		}
	}

	return bidList, nil
}

func unmarshalBid(r *bytes.Buffer, bid *Bid) error {
	if err := encoding.Read256(r, bid.X[:]); err != nil {
		return err
	}

	if err := encoding.Read256(r, bid.M[:]); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &bid.EndHeight); err != nil {
		return err
	}

	return nil
}
