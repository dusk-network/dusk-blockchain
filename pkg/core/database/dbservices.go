package database

import (
	"bytes"
	"github.com/syndtr/goleveldb/leveldb/util"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/merkletree"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

// getBlockHeaderRange gets a range of block headers in height order.
// Only returns headers that have an accompanying block.
// Note that it stops reading at the first header where this is not the case.
func (bdb *BlockchainDB) getBlockHeaderRange(start uint64, stop uint64) ([]*payload.BlockHeader, error) {
	var hdrCur *payload.BlockHeader
	var headers = make([]*payload.BlockHeader, 0, stop-start+1)
	var err error

	if stop == 0 {
		stop, err = bdb.getLatestBlockHeight()
		if err != nil {
			return nil, err
		}
	}

	var curHeight = start
	var noMoreHdrs = false
	for curHeight <= stop || noMoreHdrs {
		hdrCur, err = bdb.GetBlockHeaderByHeight(curHeight)
		if err != nil {
			return nil, err
		}
		if !bdb.blockExists(hdrCur) {
			noMoreHdrs = true
			continue
		}

		headers = append(headers, hdrCur)
		curHeight++
	}

	return headers, nil
}

// getBlockHeader gives the block header from the hash
func (bdb *BlockchainDB) getBlockHeader(hash []byte) (*payload.BlockHeader, error) {
	var header payload.BlockHeader

	table := NewTable(bdb, HEADER)
	headerBytes, err := table.Get(hash)
	if err != nil {
		return nil, err
	}

	err = header.Decode(bytes.NewReader(headerBytes))
	if err != nil {
		return nil, err
	}

	return &header, nil
}

// GetBlockHeaderByHeight gives the block header that belongs to the given block height.
func (bdb *BlockchainDB) GetBlockHeaderByHeight(height uint64) (*payload.BlockHeader, error) {
	var hash []byte
	var err error

	heightBytes := Uint64ToBytes(height)
	bhKey := append(BLOCKHEIGHT, heightBytes...)
	hash, err = bdb.Get(bhKey)
	if err != nil {
		return nil, err
	}

	hdr, err := bdb.getBlockHeader(hash)
	if err != nil {
		return nil, err
	}

	return hdr, nil
}

// getLatestHeaderHash gets the latest header hash
func (bdb *BlockchainDB) getLatestHeaderHash() ([]byte, error) {
	return bdb.Get(LATESTHEADER)
}

func (bdb *BlockchainDB) getLatestBlockHeight() (uint64, error) {
	hash, _ := bdb.getLatestHeaderHash()
	hdr, err := bdb.getBlockHeader(hash)
	if err != nil {
		return 0, err // Tricky as 0 is a valid height
	}

	return hdr.Height, nil
}

// GetLatestHeader returns the most recent header.
// Note that in this case it does not check the existence of the accompanying block.
func (bdb *BlockchainDB) GetLatestHeader() (*payload.BlockHeader, error) {
	latestHdrHash, err := bdb.getLatestHeaderHash()
	if err != nil {
		return nil, err
	}

	latestHdrKey := append(append(HEADER, latestHdrHash...))
	latestHdrBytes, err := bdb.Get(latestHdrKey)
	if err != nil {
		return nil, err
	}

	latestHdrBuf := bytes.NewReader(latestHdrBytes)
	latestHdr := &payload.BlockHeader{}
	if err = latestHdr.Decode(latestHdrBuf); err != nil {
		return nil, err
	}

	return latestHdr, nil
}

// GetBlock will return the block from the received hash
func (bdb *BlockchainDB) GetBlock(hash []byte) (*payload.Block, error) {
	var err error = nil

	block := &payload.Block{}
	table := NewTable(bdb, HEADER)

	hdrBytes, err := table.Get(hash)
	buf := bytes.NewBuffer(hdrBytes)
	hdr := &payload.BlockHeader{}
	if err = hdr.Decode(buf); err != nil {
		return nil, err
	}
	block.Header = hdr

	// If it's the Genesis block just return block without txs
	// TODO: Can this check be improved. I don't want to check at all !!
	if block.Header.Height == 0 {
		return block, nil
	}

	txHdrHashKey := append(TX, hdr.Hash...)
	txs := make([]merkletree.Payload, 0)

	// Read all the txs, first do a prefix scan on TX + header hash
	txIter := bdb.NewIterator(util.BytesPrefix(txHdrHashKey), nil)
	table = NewTable(bdb, TX)
	for txIter.Next() {
		txHashKey := txIter.Value()
		txBytes, err := table.Get(txHashKey)
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

// blockExists checks if received header has an accompanying block
func (bdb *BlockchainDB) blockExists(hdr *payload.BlockHeader) bool {
	if hdr.Height == 0 { // Genesis block always exists
		return true
	}

	txHdrHashKey := append(TX, hdr.Hash...)
	txIter := bdb.NewIterator(util.BytesPrefix(txHdrHashKey), nil)
	if txIter.First() {
		return true
	}

	return false
}
