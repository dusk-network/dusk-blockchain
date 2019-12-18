package candidate

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
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
	if err := checkHash(*b); err != nil {
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

// Make sure the hash is correct, to avoid malicious nodes from
// overwriting the candidate block for a specific hash
func checkHash(b bytes.Buffer) error {
	cm := &Candidate{}
	if err := Decode(&b, cm); err != nil {
		return err
	}

	hash := cm.Block.Header.Hash
	if err := cm.Block.SetHash(); err != nil {
		return err
	}

	if !bytes.Equal(hash, cm.Block.Header.Hash) {
		return errors.New("invalid block hash")
	}

	return nil
}
