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
	// HEADER is the prefix for a header key
	HEADER = []byte{0x01}
	// BLOCKHEIGHT is the prefix for a block height key
	BLOCKHEIGHT = []byte{0x02}
	// TX is the prefix for a transaction key
	TX = []byte{0x03}
	// UTXO is the prefix for a utxo key
	UTXO = []byte{0x04}
	// LATESTHEADER is the prefix for the latest header key (for which there is a block)
	LATESTHEADER           = []byte{0x05}
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

// WriteHeaders writes a batch of block headers to the database
func (bdb *BlockchainDB) WriteHeaders(hdrs []*payload.BlockHeader) error {
	sortedHdrs := util.SortHeadersByHeight(hdrs)

	// batchValues will consist of a header and blockheight record for each header in hdrs, plus one for the latest header
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
		heightKey := append(BLOCKHEIGHT, Uint64ToBytes(h.Height)...)
		kv[string(heightKey)] = h.Hash
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
		Infof("Writing block headers (start height=%d, end height=%d)", sortedHdrs[0].Height, sortedHdrs[len(sortedHdrs)-1].Height)

	return nil
}

// WriteBlockTransactions writes blocks to the database.
// A block transaction is linked to the header by the header hash in the transaction key
func (bdb *BlockchainDB) WriteBlockTransactions(blocks []*payload.Block) error {
	// batchValues will consist of a tx and index record for each tx in a block
	kv := make(batchValues, len(blocks)*2)

	for _, b := range blocks {

		for j, v := range b.Txs {
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
			// Key: [TX] + HEADER HASH + I <- i is the incrementer from the for loop
			// Value: TXHASH
			txHashKey := append(append(TX, b.Header.Hash...))
			txHashKey = append(txHashKey, Uint32ToBytes(uint32(j))...)

			kv[string(txHashKey)] = tx.Hash
		}
	}

	// Atomic database write of the transactions
	if err := bdb.Write(&kv); err != nil {
		log.WithField("prefix", "database").Error(errAddTransactionDbStr)
		return err
	}

	return nil
}

// ReadHeaders reads all block headers from the block chain db between start and stop hashes (start excluded, stop included)
// Although we do not return the block transactions yet, we must assure that they are there.
// If a header has no accompanying block, the reading is stopped and previously found headers are sent.
func (bdb *BlockchainDB) ReadHeaders(start []byte, stop []byte) ([]*payload.BlockHeader, error) {
	var startHeight, stopHeight = uint64(0), uint64(0)

	hdr, err := bdb.getBlockHeader(start)
	if err != nil {
		return nil, err
	}

	startHeight = hdr.Height
	if bytes.Equal(stop, make([]byte, 32)) { // Test if stop is a 32 byte hash of zeros
		hash, err := bdb.getLatestHeaderHash()
		hdr, err = bdb.getBlockHeader(hash)
		if err != nil {
			return nil, err
		}
		stopHeight = hdr.Height
	}

	// Limit to 2000 when requested more
	headersLen := stopHeight - startHeight
	if headersLen > uint64(maxRetrievableHeaders) {
		headersLen = maxRetrievableHeaders
		stopHeight = startHeight + 2000
	}
	log.WithField("prefix", "database").Debugf("Read headers from db with start height %d and stop height %d", startHeight, stopHeight)

	// Retrieve all block headers from db
	headerRange, err := bdb.getBlockHeaderRange(startHeight+1, stopHeight)
	if err != nil {
		return nil, err
	}

	return headerRange, nil
}
