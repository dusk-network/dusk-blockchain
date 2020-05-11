package user

import (
	"bytes"
)

// Bid is the 32 byte X value, created from a bidding transaction amount and M.
// FIXME: 495 - this is apparently not used anywhere
type Bid struct {
	X [32]byte
}

// Equals will return whether or not the two bids are the same.
func (b Bid) Equals(bid Bid) bool {
	return bytes.Equal(b.X[:], bid.X[:])
}
