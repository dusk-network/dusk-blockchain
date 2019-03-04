package consensus

import (
	"math/big"

	ristretto "github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/zkproof"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

func (c *Consensus) AddGenerator(bid *transactions.Bid) {
	// Set X value in publist
	dScalar := ristretto.Scalar{}
	dScalar.SetBigInt(big.NewInt(0).SetUint64(bid.Output.Amount))
	m := zkproof.BytesToScalar(bid.M)
	x := zkproof.CalculateX(dScalar, m)
	c.ctx.PubList = append(c.ctx.PubList, x)
}
