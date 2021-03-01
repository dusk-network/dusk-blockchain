// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package responding

import (
	"bytes"
	"errors"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

// BlockHashBroker is a processing unit which handles GetBlocks messages.
// It has a database connection, and a channel pointing to the outgoing message queue
// of the requesting peer.
type BlockHashBroker struct {
	db database.DB
}

// NewBlockHashBroker will return an initialized BlockHashBroker.
func NewBlockHashBroker(db database.DB) *BlockHashBroker {
	return &BlockHashBroker{
		db: db,
	}
}

// AdvertiseMissingBlocks takes a GetBlocks wire message, finds the requesting peer's
// height, and returns an inventory message of up to config.MaxInvBlocks blocks which follow the
// provided locator.
func (b *BlockHashBroker) AdvertiseMissingBlocks(srcPeerID string, m message.Message) ([]bytes.Buffer, error) {
	msg := m.Payload().(message.GetBlocks)

	// Determine from where we need to start fetching blocks, going off his Locator
	height, err := b.fetchLocatorHeight(msg)
	if err != nil {
		return nil, err
	}

	// Fill an inv message with all block hashes between the locator
	// and the chain tip.
	inv := &message.Inv{}

	for {
		height++

		var hash []byte

		err = b.db.View(func(t database.Transaction) error {
			hash, err = t.FetchBlockHashByHeight(height)
			return err
		})

		// This means we passed the tip of the chain, so we can exit the loop
		// TODO: does it really, or do we need a more sophisticated way to go about it?
		if err != nil {
			break
		}

		inv.AddItem(message.InvTypeBlock, hash)

		if len(inv.InvList) >= cfg.MaxInvBlocks {
			break
		}
	}

	// If we retrieved any items, we should marshal the inventory message, and send it
	// to the requesting peer.
	if inv.InvList != nil {
		buf, err := marshalInv(inv)
		return []bytes.Buffer{buf}, err
	}

	return nil, nil
}

// Determine a peer's height from his locator hash.
func (b *BlockHashBroker) fetchLocatorHeight(msg message.GetBlocks) (uint64, error) {
	if len(msg.Locators) == 0 {
		return 0, errors.New("empty locators array")
	}

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

func marshalInv(inv *message.Inv) (bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	if err := inv.Encode(buf); err != nil {
		return bytes.Buffer{}, err
	}

	_ = topics.Prepend(buf, topics.Inv)
	return *buf, nil
}
