package user

import (
	"bytes"
)

// Bid is the 32 byte X value, created from a bidding transaction amount and M.
// FIXME: remove the M and EndHeight as it is now managed by RUSK and we do not
// have it anymore
type Bid struct {
	X [32]byte
	//M         [32]byte
	//EndHeight uint64
}

// Equals will return whether or not the two bids are the same.
func (b Bid) Equals(bid Bid) bool {
	return bytes.Equal(b.X[:], bid.X[:])
}
