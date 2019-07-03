package consensus

import (
	"bytes"
	"encoding/binary"

	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// TODO: define a proper maxlocktime somewhere in the transactions package
const maxLockTime = 100000

func LaunchInitiator(eventBroker wire.EventBroker, rpcBus *wire.RPCBus) error {
	i, err := newInitiator(eventBroker, rpcBus)
	if err != nil {
		return err
	}

	tl := wire.NewTopicListener(eventBroker, i, string(topics.StartConsensus))
	go tl.Accept()
	i.startConsensusListener = tl
	return nil
}

type Initiator struct {
	eventBroker            wire.EventBroker
	rpcBus                 *wire.RPCBus
	db                     database.DB
	startConsensusListener *wire.TopicListener
}

func newInitiator(eventBroker wire.EventBroker, rpcBus *wire.RPCBus) (*Initiator, error) {
	drvr, err := database.From(cfg.Get().Database.Driver)
	if err != nil {
		return nil, err
	}

	db, err := drvr.Open(cfg.Get().Database.Dir, protocol.MagicFromConfig(), false)
	if err != nil {
		return nil, err
	}

	i := &Initiator{eventBroker, rpcBus, db, nil}
	return i, nil
}

func (i *Initiator) Collect(m *bytes.Buffer) error {
	// Get keys from file on disk, created by dusk-wallet
	// TODO: add this

	// Get current height
	currentHeight, err := i.getCurrentHeight()
	if err != nil {
		return err
	}

	found := i.findActiveStakes(keys, currentHeight)

	// Start listening for accepted blocks, regardless of if we found stakes or not
	acceptedBlockChan, listener := InitAcceptedBlockUpdate(i.eventBroker)

	// Unsubscribe from StartConsensus and AcceptedBlock once we're done
	defer func() {
		i.startConsensusListener.Quit()
		listener.Quit()
	}()

	if found {
		blk := <-acceptedBlockChan
		i.publishInitialRound(blk.Header.Height + 1)
		return nil
	}

	for {
		blk := <-acceptedBlockChan
		if i.keyFound(keys, blk.Txs) {
			i.publishInitialRound(blk.Header.Height + 1)
			return nil
		}
	}
}

func (i *Initiator) findActiveStakes(keys user.Keys, currentHeight uint64) bool {
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

func (i *Initiator) keyFound(keys user.Keys, txs []transactions.Transaction) bool {
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

func (i *Initiator) publishInitialRound(height uint64) {
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, height)
	i.eventBroker.Publish(msg.InitializationTopic, bytes.NewBuffer(bs))
}

func (i *Initiator) getCurrentHeight() (uint64, error) {
	req := wire.NewRequest(bytes.Buffer{}, 5)
	blkBuf, err := i.rpcBus.Call(wire.GetLastBlock, req)
	if err != nil {
		return 0, err
	}
	// We can simply get the height from the bytes at index 1 through 9.
	// That way, we dont have to decode the buffer just to get the height.
	return binary.LittleEndian.Uint64(blkBuf.Bytes()[1:9]), nil
}
