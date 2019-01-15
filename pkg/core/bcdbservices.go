package core

import (
	"bytes"
	"github.com/syndtr/goleveldb/leveldb/util"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/merkletree"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

// AddHeaders will add block headers to the database.
func (b *Blockchain) AddHeaders(msg *payload.MsgHeaders) error {
	if err := b.db.AddHeaders(msg.Headers); err != nil {
		return err
	}
	return nil
}

// GetHeaders will retrieve block headers from the database, starting and
// stopping at the provided locators.
func (b *Blockchain) GetHeaders(start []byte, stop []byte) ([]*payload.BlockHeader, error) {
	return b.db.ReadHeaders(start, stop)
}

// GetLatestHeaderHash gets the block hash of the most recent block.
func (b *Blockchain) GetLatestHeaderHash() ([]byte, error) {
	return b.db.Get(database.LATESTHEADER)
}

// GetLatestHeader will get the most recent block hash and return it as
// a block header struct.
func (b *Blockchain) GetLatestHeader() (*payload.BlockHeader, error) {
	prevHeaderHash, err := b.GetLatestHeaderHash()
	if err != nil {
		return nil, err
	}

	prevHeaderKey := append(append(database.HEADER, prevHeaderHash...))
	prevHeaderBytes, err := b.db.Get(prevHeaderKey)
	if err != nil {
		return nil, err
	}

	prevHeaderBuf := bytes.NewReader(prevHeaderBytes)
	prevHeader := &payload.BlockHeader{}
	if err := prevHeader.Decode(prevHeaderBuf); err != nil {
		return nil, err
	}

	return prevHeader, nil
}

// GetBlockByHash will return the block from the received hash
func (b *Blockchain) GetBlockByHash(hash []byte) (*payload.Block, error) {
	var err error = nil

	block := &payload.Block{}
	headerHash := append(database.HEADER, hash...)
	hdrBytes, err := b.db.Get(headerHash)
	buf := bytes.NewBuffer(hdrBytes)
	hdr := &payload.BlockHeader{}
	if err = hdr.Decode(buf); err != nil {
		return nil, err
	}
	block.Header = hdr

	txHdrHashKey := append(database.TX, hdr.Hash...)
	txs := make([]merkletree.Payload, 0)

	// Read all the txs, first do a prefix scan on TX + header hash
	txIter := b.db.NewIterator(util.BytesPrefix(txHdrHashKey), nil)
	for txIter.Next() {
		txHashKey := txIter.Value()
		txHashKey = append(database.TX, txHashKey...)
		txBytes, _ := b.db.Get(txHashKey)

		buf := bytes.NewBuffer(txBytes)
		tx := &transactions.Stealth{}
		if err = tx.Decode(buf); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	txIter.Release()

	block.Txs = txs

	return block, nil
}
