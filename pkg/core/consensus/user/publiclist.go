package user

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/zkproof"

	ristretto "github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"
)

// Bid is the 32 byte X value, created from a bidding transaction amount and M.
type Bid [32]byte

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
func (p PublicList) ValidateBids(pl PublicList, ourBid []byte) *prerror.PrError {
loop:
	for _, x := range pl {
		for _, x2 := range p {
			if bytes.Equal(x[:], x2[:]) {
				continue loop
			}
		}

		if !bytes.Equal(x[:], ourBid) {
			return prerror.New(prerror.Low, errors.New("invalid public list"))
		}
	}

	return nil
}

// GetRandomBids will get an amount of bids from the public list, to make a
// slice of scalars to be used for proof generation.
func (p PublicList) GetRandomBids(amount int) []ristretto.Scalar {
	// Shuffle the public list
	rand.Shuffle(len(p), func(i, j int) { p[i], p[j] = p[j], p[i] })

	// Create our set
	set := make([]ristretto.Scalar, amount)
	for i := 0; i < amount; i++ {
		bid := p[i][:]
		bidScalar := zkproof.BytesToScalar(bid)
		set[i] = bidScalar
	}

	return set
}

// AddBid will add a bid to the public list p.
func (p *PublicList) AddBid(bid []byte) error {
	if len(bid) != 32 {
		return fmt.Errorf("bid should be 32 bytes, is %v bytes", len(bid))
	}

	r := bytes.NewReader(bid)
	var b Bid
	if _, err := r.Read(b[:]); err != nil {
		return err
	}

	*p = append(*p, b)
	return nil
}

// RemoveBid will iterate over a public list and remove a specified bid.
func (p *PublicList) RemoveBid(bid []byte) {
	for i, b := range *p {
		if bytes.Equal(bid, b[:]) {
			list := *p
			list = append(list[:i], list[i+1:]...)
			*p = list
		}
	}
}
