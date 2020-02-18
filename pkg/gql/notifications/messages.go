package notifications

import (
	"encoding/hex"
	"encoding/json"

	"github.com/dusk-network/dusk-wallet/v2/block"
)

// BlockMsg represents the data need by Explorer UI on each new block accepted
type BlockMsg struct {
	Height    uint64
	Hash      string
	Timestamp int64
	Txs       []string

	// BlocksGeneratedCount is number of blocks generated last 24 hours
	BlocksGeneratedCount uint
}

// MarshalBlockMsg builds the JSON from a subset of block fields
func MarshalBlockMsg(blk block.Block) (string, error) {

	hash, err := blk.CalculateHash()
	if err != nil {
		return "", err
	}

	blk.Header.Hash = hash

	var p BlockMsg
	p.Height = blk.Header.Height
	p.Timestamp = blk.Header.Timestamp
	p.Hash = hex.EncodeToString(blk.Header.Hash)
	p.Txs = make([]string, 0)

	// Get a limited set of block txs hashes
	for _, tx := range blk.Txs {
		if len(p.Txs) >= maxTxsPerMsg {
			break
		}

		txid, err := tx.CalculateHash()
		if err != nil {
			return "", err
		}

		p.Txs = append(p.Txs, hex.EncodeToString(txid))
	}

	msg, err := json.Marshal(p)
	if err != nil {
		return "", err
	}

	return string(msg), nil
}
