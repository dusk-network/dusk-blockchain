package consensus

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/zkproof"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

// AddGenerator will take a bid, take out it's d and M values, and construct X.
// It then gets put into the public list.
func (c *Consensus) AddGenerator(bid transactions.Bid) {
	// Set X value in publist
	dScalar := zkproof.Uint64ToScalar(bid.Output.Amount)
	m := zkproof.BytesToScalar(bid.M)
	x := zkproof.CalculateX(dScalar, m)
	c.ctx.PubList.AddBid(x)
}
