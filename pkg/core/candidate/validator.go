package candidate

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/republisher"
	"github.com/dusk-network/dusk-wallet/v2/block"
)

// Make sure the hash and root are correct, to avoid malicious nodes from
// overwriting the candidate block for a specific hash
func Validate(m message.Message) error {
	cm := m.Payload().(message.Candidate)
	return ValidateCandidate(cm)
}

func ValidateCandidate(cm message.Candidate) error {
	if err := checkHash(cm.Block); err != nil {
		return republisher.InvalidError
	}

	if err := checkRoot(cm.Block); err != nil {
		return republisher.InvalidError
	}

	return nil
}

func checkHash(blk *block.Block) error {
	hash, err := blk.CalculateHash()
	if err != nil {
		return err
	}

	if !bytes.Equal(hash, blk.Header.Hash) {
		return errors.New("invalid block hash")
	}

	return nil
}

func checkRoot(blk *block.Block) error {
	root, err := blk.CalculateRoot()
	if err != nil {
		return err
	}

	if !bytes.Equal(root, blk.Header.TxRoot) {
		return errors.New("invalid merkle root hash")
	}

	return nil
}
