package processing

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// BlockHashBroker is a processing unit which handles GetBlocks messages.
// It has a database connection, and a channel pointing to the outgoing message queue
// of the requesting peer.
type BlockHashBroker struct {
	db           database.DB
	responseChan chan<- *bytes.Buffer
}

// NewBlockHashBroker will return an initialized BlockHashBroker.
func NewBlockHashBroker(db database.DB, responseChan chan<- *bytes.Buffer) *BlockHashBroker {
	return &BlockHashBroker{
		db:           db,
		responseChan: responseChan,
	}
}

// AdvertiseMissingBlocks takes a GetBlocks wire message, finds the requesting peer's
// height, and returns an inventory message of up to 500 blocks which follow the
// provided locator.
func (b *BlockHashBroker) AdvertiseMissingBlocks(m *bytes.Buffer) error {
	msg := &peermsg.GetBlocks{}
	if err := msg.Decode(m); err != nil {
		return err
	}

	// Determine from where we need to start fetching blocks, going off his Locator
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
		// TODO: does it really, or do we need a more sophisticated way to go about it?
		if err != nil {
			break
		}

		inv.AddItem(peermsg.InvTypeBlock, hash)
		if len(inv.InvList) >= 500 {
			break
		}
	}

	// If we retrieved any items, we should marshal the inventory message, and send it
	// to the requesting peer.
	if inv.InvList != nil {
		buf, err := marshalInv(inv)
		if err != nil {
			return err
		}

		b.responseChan <- buf
	}

	return nil
}

// Determine a peer's height from his locator hash.
func (b *BlockHashBroker) fetchLocatorHeight(msg *peermsg.GetBlocks) (uint64, error) {
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
