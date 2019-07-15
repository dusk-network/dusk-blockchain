package consensus

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/heavy"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func GetStartingRound(eventBroker wire.EventBroker, db database.DB, keys user.Keys) error {
	// Get a db connection
	if db == nil {
		_, db = heavy.CreateDBConnection()
	}

	var currentHeight uint64
	err := db.View(func(t database.Transaction) error {
		var err error
		currentHeight, err = t.FetchCurrentHeight()
		return err
	})

	if err != nil {
		currentHeight = 0
	}

	found := findActiveStakes(keys, currentHeight, db)

	// Start listening for accepted blocks, regardless of if we found stakes or not
	acceptedBlockChan, listener := InitAcceptedBlockUpdate(eventBroker)

	go func(listener *wire.TopicListener) {
		// Unsubscribe from AcceptedBlock once we're done
		defer listener.Quit()

		for {
			blk := <-acceptedBlockChan
			if found || keyFound(keys, blk.Txs) {
				roundBytes := make([]byte, 8)
				binary.LittleEndian.PutUint64(roundBytes, blk.Header.Height+1)
				eventBroker.Publish(msg.InitializationTopic, bytes.NewBuffer(roundBytes))
				return
			}
		}
	}(listener)
	return nil
}

func findActiveStakes(keys user.Keys, currentHeight uint64, db database.DB) bool {
	searchingHeight := currentHeight - transactions.MaxLockTime
	if currentHeight < transactions.MaxLockTime {
		searchingHeight = 0
	}

	for {
		var b *block.Block
		err := db.View(func(t database.Transaction) error {
			hash, err := t.FetchBlockHashByHeight(searchingHeight)
			if err != nil {
				return err
			}

			b, err = t.FetchBlock(hash)
			return err
		})

		if err != nil {
			break
		}

		if keyFound(keys, b.Txs) {
			return true
		}

		searchingHeight++
	}

	return false
}

func keyFound(keys user.Keys, txs []transactions.Transaction) bool {
	for _, tx := range txs {
		stake, ok := tx.(*transactions.Stake)
		if !ok {
			continue
		}

		if bytes.Equal(keys.BLSPubKeyBytes, stake.PubKeyBLS) {
			return true
		}
	}

	return false
}
