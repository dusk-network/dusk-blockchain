// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package genesis

import (
	"fmt"
)

// GetPresetConfig fetches a preset configuration for a genesis block for the
// given name. Returns an error, if the name does not match any existing presets.
func GetPresetConfig(name string) (Config, error) {
	c, ok := configurations[name]
	if !ok {
		return Config{}, fmt.Errorf("config not found - %s", name)
	}

	return c, nil
}

var configurations = map[string]Config{
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
