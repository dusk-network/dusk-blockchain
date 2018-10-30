package main

import (
	"flag"

	"github.com/toghrulmaharramov/dusk-go/rpc"
)

// LoadConfig will load a config file into a Config struct,
// parse command line flags if applicable and return it.
func LoadConfig() (*rpc.Config, error) {
	// Initialize config object
	cfg := rpc.Config{}

	// Parse flags and set filepath if specified
	flag.Parse()
	if *conf != "dusk.conf" {
		cfg.File = *conf
	}

	// Create config file if it doesn't exist yet
	if err := cfg.LoadConfig(); err != nil {
		return nil, err
	}

	// Override config options with flags if specified
	if *rpcuser != "dusk123" {
		cfg.RPCUser = *rpcuser
	}
	if *rpcpass != "duskpass" {
		cfg.RPCPass = *rpcpass
	}
	if *rpcport != "9999" {
		cfg.RPCPort = *rpcport
	}

	return &cfg, nil
}
