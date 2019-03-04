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

// PublicList is a list of bid X values.
type PublicList []Bid

// CreatePubList will turn a slice of bytes into a PublicList.
func CreatePubList(pl []byte) (PublicList, *prerror.PrError) {
	if len(pl)%32 != 0 {
		return nil, prerror.New(prerror.Low, errors.New("malformed public list"))
	}

	numBids := len(pl) / 32
	r := bytes.NewReader(pl)
	pubList := make(PublicList, numBids)
	for i := 0; i < numBids; i++ {
		var bid Bid
		if _, err := r.Read(bid[:]); err != nil {
			return nil, prerror.New(prerror.High, err)
		}

		pubList[i] = bid
	}

	return pubList, nil
}

// ValidateBids will check if pl contains valid bids.
func (p PublicList) ValidateBids(pl PublicList, ourBid Bid) *prerror.PrError {
loop:
	for _, x := range pl {
		for _, x2 := range p {
			if x.Equals(x2) {
				continue loop
			}
		}

		if !x.Equals(ourBid) {
			return prerror.New(prerror.Low, errors.New("invalid public list"))
		}
	}

	return nil
}

// GetRandomBids will get an amount of bids from the public list, to make a
// slice of scalars to be used for proof generation.
func (p PublicList) GetRandomBids(amount int) []Bid {
	// Shuffle the public list
	rand.Shuffle(len(p), func(i, j int) { p[i], p[j] = p[j], p[i] })

	// Create our set
	set := make([]Bid, amount)
	for i := 0; i < amount; i++ {
		set[i] = p[i]
	}

	return set
}

// AddBid will add a bid to the public list p.
func (p *PublicList) AddBid(bid Bid) {
	// Check for duplicates
	for _, b := range *p {
		if b.Equals(bid) {
			return
		}
	}

	*p = append(*p, bid)
}

// RemoveBid will iterate over a public list and remove a specified bid.
func (p *PublicList) RemoveBid(bid Bid) {
	for i, b := range *p {
		if b.Equals(bid) {
			list := *p
			list = append(list[:i], list[i+1:]...)
			*p = list
		}
	}
}
