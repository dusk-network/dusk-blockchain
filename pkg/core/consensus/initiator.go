package consensus

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// TODO: define a proper maxlocktime somewhere in the transactions package
const maxLockTime = 100000

type initiator struct {
	db        database.DB
	publisher wire.EventPublisher
}

func newInitiator(db database.DB, publisher wire.EventPublisher) *initiator {
	return &initiator{db, publisher}
}

func Initiate(eventBroker wire.EventBroker, keys user.Keys, db database.DB, currentHeight uint64) {
	initiator := newInitiator(db, eventBroker)

	acceptedBlockChan := InitAcceptedBlockUpdate(eventBroker)
	if initiator.foundActiveStakes(keys, currentHeight) {
		blk := <-acceptedBlockChan

		initiator.publishInitialRound(blk.Header.Height + 1)
		return
	}

	for {
		blk := <-acceptedBlockChan

		if initiator.keyFound(keys, blk.Txs) {
			initiator.publishInitialRound(blk.Header.Height + 1)
			return
		}
	}
}

func (i *initiator) foundActiveStakes(keys user.Keys, currentHeight uint64) bool {
	searchingHeight := currentHeight - maxLockTime
	if currentHeight < maxLockTime {
		searchingHeight = 0
	}

	for {
		var b *block.Block
		err := i.db.View(func(t database.Transaction) error {
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

		if i.keyFound(keys, b.Txs) {
			return true
		}

		searchingHeight++
	}

	return false
}

func (i *initiator) keyFound(keys user.Keys, txs []transactions.Transaction) bool {
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

func (i *initiator) publishInitialRound(height uint64) {
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, height)
	i.publisher.Publish(msg.InitializationTopic, bytes.NewBuffer(bs))
}
