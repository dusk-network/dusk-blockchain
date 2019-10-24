package generation

import (
	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-wallet/transactions"
	log "github.com/sirupsen/logrus"
)

// Use the TxRetriever to get a valid bid transaction that belongs to us, and return the D value from that bid.
func getD(m ristretto.Scalar, subscriber eventbus.Subscriber, db database.DB) ristretto.Scalar {
	retriever := newBidRetriever(db)
	bid, err := retriever.SearchForBid(m.Bytes())
	if err != nil {
		log.WithField("process", "generation").Debugln("no bids belonging to us found in the chain")

		// If we did not get any values from scanning the chain, we will wait to get a valid one from incoming blocks
		bid = waitForBid(subscriber, m.Bytes())

		hash, err := bid.CalculateHash()
		if err != nil {
			// If we found a valid bid tx, we should under no circumstance have issues marshalling it
			panic(err)
		}

		log.WithFields(log.Fields{
			"process": "generation",
			"tx hash": hash,
		}).Debugln("new bid received")
	}

	d := ristretto.Scalar{}
	d.UnmarshalBinary(bid.(*transactions.Bid).Outputs[0].Commitment.Bytes())
	return d
}

func waitForBid(subscriber eventbus.Subscriber, m []byte) transactions.Transaction {
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
