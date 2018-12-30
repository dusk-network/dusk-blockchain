package database

import (
	"bytes"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

// getBlockHeaderRange gets a range of block headers
func getBlockHeaderRange(db *BlockchainDB, start uint64, stop uint64) ([]*payload.BlockHeader, error) {
	var bhCur *payload.BlockHeader
	var headers = make([]*payload.BlockHeader, stop-start+1)
	var err error

	if stop == 0 {
		bhCur, err = getBlockHeaderByHeight(db, stop)
		if err != nil {
			return nil, err
		}
		stop = bhCur.Height
	}

	var curHeight = start
	for start <= stop {
		bhCur, err = getBlockHeaderByHeight(db, curHeight)
		if err != nil {
			return nil, err
		}

		headers = append(headers, bhCur)
		curHeight++
	}

	return headers, nil
}

// getBlockHeaderByHeight gives the block header that belongs to the given block height.
// It will return the latest block header if height is 0.
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

// Get the latest header
func getLatestBlockHash(db *BlockchainDB) ([]byte, error) {
	hash, err := db.Get(LATESTHEADER)
	if err != nil {
		return nil, err
	}

	return hash, nil
}
