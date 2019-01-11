package database

import (
	"bytes"
	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/util"
)

// BlockchainDB holds a database interface and the path to the physical db dir.
type BlockchainDB struct {
	Database
	path string
}

const maxRetrievableHeaders = 2000

var (
	// TX is the prefix for a transaction key
	TX = []byte{0x01}
	// BLOCKHEIGHT is the prefix for a block height key
	BLOCKHEIGHT = []byte{0x02}
	// HEADER is the prefix for a header key
	HEADER = []byte{0x03}
	// LATESTHEADER is the prefix for the latest header key
	LATESTHEADER = []byte{0x04}
	// UTXO is the prefix for a utxo key
	UTXO = []byte{0x05}

	errAddBlockHeaderDbStr = "Failed to add block header to db."
	errAddTransactionDbStr = "Failed to add transaction to db."
)

// NewBlockchainDB returns a pointer to a newly created or existing blockchain db.
func NewBlockchainDB(path string) (*BlockchainDB, error) {
	db, err := NewDatabase(path)
	if err != nil {
		return nil, err
	}

	return &BlockchainDB{db, path}, nil
}

// AddHeaders adds a batch of block headers to the database
func (bdb *BlockchainDB) AddHeaders(hdrs []*payload.BlockHeader) error {

	sortedHdrs := util.SortHeadersByHeight(hdrs)

	// batch will consist of a header and blockheight record for each header in hdrs, plus one for the latest header
	kv := make(batchValues, len(sortedHdrs)*2+1)

	// This is the main mapping
	// Key: HEADER + blockheader hash, Value: blockheader bytes
	for _, h := range sortedHdrs {
		b, _ := h.Bytes()
		headerHash := append(HEADER, h.Hash...)
		kv[string(headerHash)] = b
	}

	// This is the secondary mapping
	// Key: BLOCKHEIGHT + block height, Value: blockhash
	for _, h := range sortedHdrs {
		heightBytes := append(BLOCKHEIGHT, Uint64ToBytes(h.Height)...)
		kv[string(heightBytes)] = h.Hash
	}

	// This is the third mapping
	kv[string(LATESTHEADER)] = sortedHdrs[len(sortedHdrs)-1].Hash

	err := bdb.Write(&kv)
	if err != nil {
		log.WithField("prefix", "database").
			Errorf(errAddBlockHeaderDbStr+" (start height=%d, end height=%d)", sortedHdrs[0].Height, sortedHdrs[len(sortedHdrs)-1].Height)
		return err
	}

	log.WithField("prefix", "database").
		Infof("Adding block headers (start height=%d, end height=%d)", sortedHdrs[0].Height, sortedHdrs[len(sortedHdrs)-1].Height)
	//	Info("Adding block headers (start height=%d, end height=%d)", uint64(sortedHdrs[0].Height), uint64(sortedHdrs[len(sortedHdrs)-1].Height))

	return nil
}

// AddBlockTransactions adds a block to the database
// A transaction is linked to the block by its hash in the key
func (bdb *BlockchainDB) AddBlockTransactions(block *payload.Block) error {
	//TODO: add batch of transactions so we can use database batch writes

	// batch will consist of a tx and index record for each tx in txs
	kv := make(batchValues, len(block.Txs)*2)

	for i, v := range block.Txs {
		tx := v.(*transactions.Stealth)
		buf := new(bytes.Buffer)
		err := tx.Encode(buf)
		if err != nil {
			log.WithField("prefix", "database").Errorf(errAddTransactionDbStr, tx)
			return err
		}
		// This is the main mapping
		// Key: [TX] + TXHASH
		txBytes := buf.Bytes()
		txKey := append(TX, tx.Hash...)
		kv[string(txKey)] = txBytes

		// This is the index
		// Key: [TX] + BLOCKHASH + I <- i is the incrementer from the for loop
		// Value: TXHASH
		blockhashKey := append(append(TX, block.Header.Hash...))
		blockhashKey = append(blockhashKey, Uint32ToBytes(uint32(i))...)

		kv[string(blockhashKey)] = tx.Hash
	}

	err := bdb.Write(&kv)
	if err != nil {
		log.WithField("prefix", "database").Error(errAddTransactionDbStr)
		return err
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
