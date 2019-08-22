package generation

import (
	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
)

// Use the TxRetriever to get a valid bid transaction that belongs to us, and return the D value from that bid.
func getD(m ristretto.Scalar, subscriber wire.EventSubscriber, db database.DB) ristretto.Scalar {
	retriever := newBidRetriever(db)
	bid, err := retriever.SearchForBid(m.Bytes())
	if err != nil {
		// If we did not get any values from scanning the chain, we will wait to get a valid one from incoming blocks
		bid = waitForBid(subscriber, m.Bytes())
	}

	d := ristretto.Scalar{}
	d.UnmarshalBinary(bid.(*transactions.Bid).Outputs[0].Commitment)
	return d
}

func waitForBid(subscriber wire.EventSubscriber, m []byte) transactions.Transaction {
	acceptedBlockChan, listener := consensus.InitAcceptedBlockUpdate(subscriber)
	defer listener.Quit()

	for {
		blk := <-acceptedBlockChan
		bid, err := findCorrespondingBid(blk.Txs, m, 0, 0)
		if err != nil {
			continue
		}

		return bid
	}
}
