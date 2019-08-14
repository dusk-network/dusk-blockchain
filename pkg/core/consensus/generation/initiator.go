package generation

import (
	"bytes"
	"errors"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/initiation"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	zkproof "github.com/dusk-network/dusk-zkproof"
)

func getLatestBid(k ristretto.Scalar, subscriber wire.EventSubscriber, db database.DB) ristretto.Scalar {
	// Get our M value to compare
	m := zkproof.CalculateM(k)

	initiator := initiation.NewInitiator(db, m.Bytes(), FindD)
	dBytes, err := initiator.SearchForValue()
	if err != nil {
		// If we did not get any values from scanning the chain, we will wait to get a valid one from incoming blocks
		dBytes = waitForBid(subscriber, m.Bytes())
	}

	d := ristretto.Scalar{}
	d.UnmarshalBinary(dBytes)
	return d
}

func FindD(txs []transactions.Transaction, item []byte) ([]byte, error) {
	for _, tx := range txs {
		bid, ok := tx.(*transactions.Bid)
		if !ok {
			continue
		}

		if bytes.Equal(item, bid.M) {
			return bid.Outputs[0].Commitment, nil
		}
	}

	return nil, errors.New("could not find a corresponding d value")
}

func waitForBid(subscriber wire.EventSubscriber, m []byte) []byte {
	acceptedBlockChan, listener := consensus.InitAcceptedBlockUpdate(subscriber)
	defer listener.Quit()

	for {
		blk := <-acceptedBlockChan
		dBytes, err := FindD(blk.Txs, m)
		if err != nil {
			continue
		}

		return dBytes
	}
}
