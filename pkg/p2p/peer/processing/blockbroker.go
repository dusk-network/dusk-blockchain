package processing

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

type BlockBroker struct {
	db           database.DB
	responseChan chan<- *bytes.Buffer
}

func NewBlockBroker(db database.DB, responseChan chan<- *bytes.Buffer) *BlockBroker {
	return &BlockBroker{
		db:           db,
		responseChan: responseChan,
	}
}

// Send back the set of block hashes between the message locator and target.
func (b *BlockBroker) AdvertiseMissingBlocks(m *bytes.Buffer) error {
	msg := &peermsg.GetBlocks{}
	if err := msg.Decode(m); err != nil {
		return err
	}

	// Fetch blocks to send to peer, going off his Locator
	height, err := b.fetchLocatorHeight(msg)
	if err != nil {
		return err
	}

	// Fill an inv message with all block hashes between the locator
	// and the chain tip.
	inv := &peermsg.Inv{}
	for {
		height++

		var hash []byte
		err := b.db.View(func(t database.Transaction) error {
			hash, err = t.FetchBlockHashByHeight(height)
			return err
		})

		// This means we passed the tip of the chain, so we can exit the loop
		if err != nil {
			break
		}

		inv.AddItem(peermsg.InvTypeBlock, hash)
		if len(inv.InvList) >= 500 {
			break
		}
	}

	if inv.InvList != nil {
		buf, err := marshalInv(inv)
		if err != nil {
			return err
		}

		b.responseChan <- buf
	}

	return nil
}

func (b *BlockBroker) fetchLocatorHeight(msg *peermsg.GetBlocks) (uint64, error) {
	var height uint64
	err := b.db.View(func(t database.Transaction) error {
		header, err := t.FetchBlockHeader(msg.Locators[0])
		if err != nil {
			return err
		}

		height = header.Height
		return nil
	})

	return height, err
}

func marshalInv(inv *peermsg.Inv) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	if err := inv.Encode(buf); err != nil {
		return nil, err
	}

	return wire.AddTopic(buf, topics.Inv)
}
