// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package conf

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Registry holds General and RPC
type Registry struct {
	General generalConfiguration
	RPC     rpcConfiguration
}

// Node general configs
type generalConfiguration struct {
	Network string
}

// rpc client configs
type rpcConfiguration struct {
	Network string
	Address string

	EnableTLS bool
	CertFile  string
	KeyFile   string
	Hostname  string

	User string
	Pass string
}

// InitConfig will init the config vars from viper
func InitConfig(cfg string) Registry {
	viper.SetConfigName("dusk")
	if cfg == "" {
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.dusk/")
	} else {
		viper.AddConfigPath(cfg)
	}

	if err := viper.ReadInConfig(); err != nil {
		// FIXME: 493 - this is completely outdated. We do not use dusk-wallet-cli
		// anymore
		_, _ = fmt.Fprintln(os.Stdout, "Config file not found. Please place dusk-wallet-cli in the same directory as your dusk.toml file.")
		os.Exit(0)
	}

	var conf Registry
	if err := viper.Unmarshal(&conf); err != nil {
		_, _ = fmt.Fprintln(os.Stdout, "Could not decode config file "+err.Error())
		os.Exit(1)
	}

	return conf
}
