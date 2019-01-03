package database

import (
	"bytes"
	"fmt"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

type BlockchainDB struct {
	Database
	path string
}

const maxRetrievableHeaders = 2000

var (
	// TX, HEADER AND UTXO are the prefixes for the db
	TX           = []byte{0x01}
	BLOCKHEIGHT  = []byte{0x02}
	HEADER       = []byte{0x03}
	LATESTHEADER = []byte{0x04}
	UTXO         = []byte{0x05}
)

func NewBlockchainDB(path string) (*BlockchainDB, error) {
	db, err := NewDatabase(path)
	if err != nil {
		return nil, err
	}

	return &BlockchainDB{db, path}, nil
}

func (bdb *BlockchainDB) AddHeader(header *payload.BlockHeader) error {
	headerTable := NewTable(bdb, HEADER)

	fmt.Printf("Adding block header (height=%d)\n", header.Height)

	// This is the main mapping
	// Key: HEADER + blockheader hash, Value: blockheader bytes
	hashKey := header.Hash
	headerBytes, err := header.Bytes()
	headerTable.Put(hashKey, headerBytes)

	// This is the secondary mapping
	// Key: HEADER + block height, Value: blockhash
	heightTable := NewTable(bdb, BLOCKHEIGHT)
	heightBytes := Uint64ToBytes(header.Height)
	err = heightTable.Put(heightBytes, hashKey)
	if err != nil {
		return err
	}
	// This is the third mapping
	// WARNING: This assumes that headers are adding in order.
	latestHeaderTable := NewTable(bdb, []byte{})
	return latestHeaderTable.Put(LATESTHEADER, header.Hash)
}

func (bdb *BlockchainDB) AddTransactions(blockhash []byte, txs []transactions.Stealth) error {

	// SHOULD BE DONE IN BATCH!!!!
	for i, tx := range txs {
		buf := new(bytes.Buffer)
		fmt.Println(tx.Hash)
		err := tx.Encode(buf)
		if err != nil {
			fmt.Println("Error adding transaction with bytes ", tx)
			return err
		}
		txBytes := buf.Bytes()
		txhash := tx.Hash

		// This is the original mapping
		// Key: [TX] + TXHASH
		txKey := append(TX, txhash...)
		err = bdb.Put(txKey, txBytes)
		if err != nil {
			fmt.Println("Error could not add tx into db ", txBytes)
			return err
		}

		// This is the index
		// Key: [TX] + BLOCKHASH + I <- i is the incrementer from the for loop
		// Value: TXHASH
		blockhashKey := append(append(TX, blockhash...))
		blockhashKey = append(blockhashKey, Uint32ToBytes(uint32(i))...)

		err = bdb.Put(blockhashKey, txhash)
		if err != nil {
			fmt.Println("Error could not add tx index into db")
			return err
		}
	}
	return nil
}

// ReadHeaders reads all block headers from the block chain db between start and stop hashes (start excluded, stop included)
func (bdb *BlockchainDB) ReadHeaders(start []byte, stop []byte) ([]*payload.BlockHeader, error) {
	var startHeight, stopHeight = uint64(0), uint64(0)

	startHeight, err := getHeightByBlockHash(bdb, start)
	if err != nil {
		return nil, err
	}
	//
	if bytes.Equal(stop, []byte{}) {
		hash, err := getLatestBlockHash(bdb)
		stopHeight, err = getHeightByBlockHash(bdb, hash)
		if err != nil {
			return nil, err
		}
	}

	// Limit to 2000 when requested more
	headersLen := stopHeight - startHeight
	if headersLen > maxRetrievableHeaders {
		headersLen = maxRetrievableHeaders
		stopHeight = startHeight + 1 + 2000
	}

	// Retrieve all block headers from db
	headers := make([]*payload.BlockHeader, headersLen)
	headerRange, err := getBlockHeaderRange(bdb, startHeight+1, stopHeight)
	if err != nil {
		return nil, err
	}
	// Add the retrieved range of block headers
	headers = append(headers, headerRange...)

	return headers, nil
}
