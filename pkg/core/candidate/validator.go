package candidate

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/republisher"
)

// Validate complies with the republisher.Validator interface. It internally invokes ValidateCandidate
func Validate(m message.Message) error {
	cm := m.Payload().(message.Candidate)
	return ValidateCandidate(cm)
}

// ValidateCandidate makes sure the hash and root are correct, to avoid malicious nodes from // overwriting the candidate block for a specific hash
func ValidateCandidate(cm message.Candidate) error {
	if err := checkHash(cm.Block); err != nil {
		log.WithError(err).Errorln("validation failed")
		return republisher.InvalidError
	}

	if err := checkRoot(cm.Block); err != nil {
		log.WithError(err).Errorln("validation failed")
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
