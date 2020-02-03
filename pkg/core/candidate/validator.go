package candidate

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/republisher"
	"github.com/dusk-network/dusk-wallet/block"
)

// Make sure the hash and root are correct, to avoid malicious nodes from
// overwriting the candidate block for a specific hash
func Validate(buf bytes.Buffer) error {
	cm := &Candidate{block.NewBlock(), block.EmptyCertificate()}
	if err := Decode(&buf, cm); err != nil {
		return republisher.EncodingError
	}

	if err := checkHash(cm.Block); err != nil {
		return republisher.InvalidError
	}

	if err := checkRoot(cm.Block); err != nil {
		return republisher.InvalidError
	}

	return nil
}

func checkHash(blk *block.Block) error {
	hash := make([]byte, 32)
	copy(hash, blk.Header.Hash)
	if err := blk.SetHash(); err != nil {
		return err
	}

	if !bytes.Equal(hash, blk.Header.Hash) {
		return errors.New("invalid block hash")
	}

	return nil
}

func checkRoot(blk *block.Block) error {
	root := make([]byte, 32)
	copy(root, blk.Header.TxRoot)
	if err := blk.SetRoot(); err != nil {
		return err
	}

	if !bytes.Equal(root, blk.Header.TxRoot) {
		return errors.New("invalid merkle root hash")
	}

	return nil
}
