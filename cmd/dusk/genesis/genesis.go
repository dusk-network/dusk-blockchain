// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package genesis

import (
	"encoding/json"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/config/genesis"
	"github.com/urfave/cli"
)

// Action prints a genesis.
func Action(c *cli.Context) error {
	blk := genesis.Decode()

	b, err := json.MarshalIndent(blk, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(b))
	return nil
}
