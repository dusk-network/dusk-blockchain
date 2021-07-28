// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package main

import (
	"flag"
)

var configToml = flag.String("config", "", "dusk.toml")

func main() {
	flag.Parse()

	d := Deployer{ConfigPath: *configToml}
	d.Run()
}
