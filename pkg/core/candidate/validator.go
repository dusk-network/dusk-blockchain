// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package candidate

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
)

// Validate an incoming Candidate message.
func Validate(m message.Message) ([]byte, error) {
	cm := m.Payload().(block.Block)
	return SanityCheckCandidate(cm)
}

// SanityCheckCandidate makes sure the hash is correct, to avoid
// malicious nodes from overwriting the candidate block for a specific hash.
func SanityCheckCandidate(cm block.Block) ([]byte, error) {
	var (
		hash []byte
		err  error
	)

	if hash, err = checkHash(&cm); err != nil {
		log.WithError(err).Errorln("validation failed")
		return nil, errors.New("invalid candidate received")
	}

	return hash, nil
}

func checkHash(blk *block.Block) ([]byte, error) {
	hash, err := blk.CalculateHash()
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(hash, blk.Header.Hash) {
		return nil, errors.New("invalid block hash")
	}

	return hash, nil
}
