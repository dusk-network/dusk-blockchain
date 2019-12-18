package candidate

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-wallet/block"
)

// The candidate validator performs a simple integrity check with an
// incoming block's hash, before republishing it.
type validator struct {
	publisher eventbus.Publisher
}

func newValidator(pub eventbus.Publisher) *validator {
	return &validator{pub}
}

func (v *validator) Process(b *bytes.Buffer) error {
	if err := checkHashAndRoot(*b); err != nil {
		return err
	}

	return v.republish(*b)
}

func (v *validator) republish(b bytes.Buffer) error {
	if err := topics.Prepend(&b, topics.Candidate); err != nil {
		return err
	}

	v.publisher.Publish(topics.Gossip, &b)
	return nil
}

// Make sure the hash and root are correct, to avoid malicious nodes from
// overwriting the candidate block for a specific hash
func checkHashAndRoot(b bytes.Buffer) error {
	cm := &Candidate{block.NewBlock(), block.EmptyCertificate()}
	if err := Decode(&b, cm); err != nil {
		return err
	}

	if err := checkHash(cm.Block); err != nil {
		return err
	}

	return checkRoot(cm.Block)
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
