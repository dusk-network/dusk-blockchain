// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package genesis

import (
	"errors"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
)

// Config is a list of configuration parameters for generating a genesis block.
type Config struct {
	timestamp int64
	seed      []byte

	Transactions []transactions.ContractCall
}

// NewConfig will construct a new genesis config. This function does sanity checks
// on all passed parameters, and ensures a correct genesis block can be produced
// from them.
func NewConfig(timestamp int64, seed []byte, txs []transactions.ContractCall) (Config, error) {
	if timestamp > time.Now().Unix() {
		return Config{}, errors.New("can not generate genesis blocks from the future")
	}

	c := Config{
		timestamp:    timestamp,
		seed:         seed,
		Transactions: txs,
	}

	return c, nil
}
