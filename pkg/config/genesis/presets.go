// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package genesis

import (
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
)

// GetPresetConfig fetches a preset configuration for a genesis block for the
// given name. Returns an error, if the name does not match any existing presets.
func GetPresetConfig(name string) (Config, error) {
	c, ok := configurations[name]
	if !ok {
		return Config{}, fmt.Errorf("config not found - %s", name)
	}

	return addDefaultTxs(name, c), nil
}

var configurations = map[string]Config{
	"testnet": {
		// March 9, 2022 16:10:22 GMT
		timestamp: 1646842222,
		seed:      make([]byte, 33),
		hash:      config.TESTNET_GENESIS_HASH,
	},
	"devnet": {
		// March 9, 2022 16:10:22 GMT
		timestamp: 1646842222,
		seed:      make([]byte, 33),
	},
	"stressnet": {
		// March 9, 2022 16:10:22 GMT
		timestamp: 1646842222,
		seed:      make([]byte, 33),
	},
}

func addDefaultTxs(name string, c Config) Config {
	c.Transactions = make([]transactions.ContractCall, 0)

	// unit tests are using devnet genesis and they need a mocked transaction. On
	// the other hand, in order to be aligned with default testnet genesis hash
	// we need one empty tx in testnet genesis block.
	switch name {
	case "testnet":
		c.Transactions = append(c.Transactions, transactions.EmptyTx())
	default:
		c.Transactions = append(c.Transactions, transactions.MockTx())
	}

	return c
}
