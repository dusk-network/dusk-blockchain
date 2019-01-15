package database

import (
	"bytes"
	"github.com/syndtr/goleveldb/leveldb/util"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

// getBlockHeaderRange gets a range of block headers
func getBlockHeaderRange(db *BlockchainDB, start uint64, stop uint64) ([]*payload.BlockHeader, error) {
	var hdrCur *payload.BlockHeader
	var headers = make([]*payload.BlockHeader, 0, stop-start+1)
	var err error

	if stop == 0 {
		hdrCur, err = getBlockHeaderByHeight(db, stop)
		if err != nil {
			return nil, err
		}
		stop = hdrCur.Height
	}

	var curHeight = start
	var noMoreHdrs = false
	for curHeight <= stop || noMoreHdrs {
		hdrCur, err = getBlockHeaderByHeight(db, curHeight)
		if err != nil {
			return nil, err
		}
		// Stop acquiring headers if the last call gave none
		if hdrCur == nil {
			noMoreHdrs = true
			continue
		}

		headers = append(headers, hdrCur)
		curHeight++
	}

	return headers, nil
}

// getBlockHeaderByHeight gives the block header that belongs to the given block height.
// It will return the latest block header if height is 0.
// Note: only the header of a complete block is returned, so a block with transactions.
func getBlockHeaderByHeight(db *BlockchainDB, height uint64) (*payload.BlockHeader, error) {
	var hash []byte
	var header payload.BlockHeader
	var err error

	if height != 0 {
		heightBytes := Uint64ToBytes(height)
		bhKey := append(BLOCKHEIGHT, heightBytes...)
		hash, err = db.Get(bhKey)
		if err != nil {
			return nil, err
		}
	} else {
		// Get the latest hash
		hash, err = getLatestBlockHash(db)
		if err != nil {
			return nil, err
		}
	}

	hKey := append(HEADER, hash...)
	headerBytes, err := db.Get(hKey)
	if err != nil {
		return nil, err
	}

	err = header.Decode(bytes.NewReader(headerBytes))
	if err != nil {
		return nil, err
	}

	// Check if header belongs to existing block
	txHdrHashKey := append(TX, header.Hash...)
	txIter := db.NewIterator(util.BytesPrefix(txHdrHashKey), nil)
	if height != 0 && !txIter.First() {
		return nil, nil
	}

	return &header, nil
}

func getHeightByBlockHash(db *BlockchainDB, hash []byte) (uint64, error) {
	var header payload.BlockHeader

	hKey := append(HEADER, hash...)
	headerBytes, err := db.Get(hKey)
	if err != nil {
		return 0, err
	}
	err = header.Decode(bytes.NewReader(headerBytes))
	if err != nil {
		return 0, err
	}

	return header.Height, nil
}

// Get the latest block hash
func getLatestBlockHash(db *BlockchainDB) ([]byte, error) {
	hash, err := db.Get(LATESTHEADER)
	if err != nil {
		return nil, err
	}

	return hash, nil
}
