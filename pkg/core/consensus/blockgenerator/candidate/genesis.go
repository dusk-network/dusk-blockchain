// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package candidate

import (
	"bytes"
	"context"
	"encoding/hex"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

// GenerateGenesisBlock is a developer utility for regenerating the genesis block
// as they would be different per network type. Once a genesis block is
// approved, its hex blob should be copied into config.TestNetGenesisBlob.
func GenerateGenesisBlock(e *consensus.Emitter) (string, error) {
	f := func(ctx context.Context, txs []transactions.ContractCall, bh uint64, gasLimit uint64, generator []byte) ([]transactions.ContractCall, []byte, error) {
		return txs, make([]byte, 32), nil
	}

	g := &generator{
		Emitter:   e,
		executeFn: f,
	}

	// TODO: do we need to generate correct proof and score
	seed, _ := crypto.RandEntropy(33)

	b, err := g.GenerateBlock(0, seed, make([]byte, 32), time.Now().Unix(), [][]byte{{0}})
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	if err := message.MarshalBlock(buf, b); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf.Bytes()), nil
}
