package generation

import (
	"bytes"
	"errors"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	zkproof "github.com/dusk-network/dusk-zkproof"
)

func getLatestBid(k ristretto.Scalar, subscriber wire.EventSubscriber, db database.DB) ristretto.Scalar {
	// Get our M value to compare
	m := zkproof.CalculateM(k)

	retriever := consensus.NewTxRetriever(db, FindD)
	bid, err := retriever.SearchForTx(m.Bytes())
	if err != nil {
		// If we did not get any values from scanning the chain, we will wait to get a valid one from incoming blocks
		bid = waitForBid(subscriber, m.Bytes())
	}

	d := ristretto.Scalar{}
	d.UnmarshalBinary(bid.(*transactions.Bid).Outputs[0].Commitment)
	return d
}

// TODO: find a way to keep this unexported, as it is currently only public for a test.
func FindD(txs []transactions.Transaction, item []byte) (transactions.Transaction, error) {
	for _, tx := range txs {
		bid, ok := tx.(*transactions.Bid)
		if !ok {
			continue
		}

		if bytes.Equal(item, bid.M) {
			return bid, nil
		}
	}

	return nil, errors.New("could not find a corresponding d value")
}

func waitForBid(subscriber wire.EventSubscriber, m []byte) transactions.Transaction {
	acceptedBlockChan, listener := consensus.InitAcceptedBlockUpdate(subscriber)
	defer listener.Quit()

	for {
		blk := <-acceptedBlockChan
		bid, err := FindD(blk.Txs, m)
		if err != nil {
			continue
		}

		return bid
	}
}
