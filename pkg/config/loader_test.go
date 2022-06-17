// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package config

import (
	"os"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	defaultDuskConfig = "--config=./samples/default.dusk.toml"
	customConfig      = "--config=./samples/custom.toml"
)

var initialArgs = os.Args

// Changes in default.dusk.toml might break this test.
func TestDefaultConfigTOML(t *testing.T) {
	Reset()

	// This relies on default.dusk.toml
	// Mock command line arguments
	os.Args = append(os.Args, defaultDuskConfig)

	if err := Load("default.dusk", nil, nil); err != nil {
		t.Errorf("Failed parse: %v", err)
	}

	if Get().General.Network != "devnet" { //nolint
		t.Error("Invalid general/network value")
	}

	if Get().Logger.Level != "debug" { //nolint
		t.Error("Invalid logger level")
	}

	if Get().State.BlockGasLimit != DefaultBlockGasLimit { //nolint
		t.Errorf("Invalid block gas limit: %d", Get().State.BlockGasLimit)
	}
}

// TestSupportedFlags to ensure all supported flags are properly bound and they
// overwrite the values loaded from the config file.
func TestSupportedFlags(t *testing.T) {
	Reset()

	// Mock command line arguments
	os.Args = append(os.Args, defaultDuskConfig)
	// Ensure here to list all supported CLI flags
	os.Args = append(os.Args, "--logger.level=custom")
	os.Args = append(os.Args, "--general.network=mainnet")
	os.Args = append(os.Args, "--network.port=9876")
	os.Args = append(os.Args, "--logger.output=modified")

	// This relies on default.dusk.toml
	if err := Load("default.dusk", nil, nil); err != nil {
		t.Errorf("Failed parse: %v", err)
	}

	if Get().Logger.Level != "custom" {
		t.Errorf("Invalid logger level %s", Get().Logger.Level)
	}

	if Get().General.Network != "mainnet" {
		t.Errorf("Invalid network value %s", Get().General.Network)
	}

	if Get().Logger.Output != "modified" {
		t.Errorf("Invalid logger output %s", Get().Logger.Output)
	}

	if Get().Network.Port != "9876" {
		t.Errorf("Invalid network port %s", Get().Network.Port)
	}
}

// TestSupportedEnv
//
// Ensures all supported ENV variables are properly bound and they overwrite the
// values loaded from the config file.
//
// Ensures all supported EVN have lower priority than CLI flags.
func TestSupportedEnv(t *testing.T) {
	Reset()

	// Mock command line arguments
	os.Args = append(os.Args, defaultDuskConfig)

	os.Setenv("DUSK_GENERAL_NETWORK", "GLOBAL_VAR")
	os.Setenv("DUSK_LOGGER_LEVEL", "NEW_LEVEL")
	viper.AutomaticEnv()

	// This relies on default.dusk.toml
	if err := Load("default.dusk", nil, nil); err != nil {
		t.Errorf("Failed parse: %v", err)
	}

	if Get().General.Network != "GLOBAL_VAR" {
		t.Errorf("Invalid ENV value: %s", Get().General.Network)
	}

	if Get().Logger.Level != "NEW_LEVEL" {
		t.Errorf("Invalid Logger ENV value %s", Get().Logger.Level)
	}
}

func TestReadOnly(t *testing.T) {
	Reset()

	// Mock command line arguments
	os.Args = append(os.Args, defaultDuskConfig)

	// This relies on default.dusk.toml
	if err := Load("default.dusk", nil, nil); err != nil {
		t.Errorf("Failed parse: %v", err)
	}

	if Get().Logger.Level != "debug" {
		t.Error("Invalid logger level")
	}

	r := Get()
	r.Logger.Level = "MODIFIED_level"

	if Get().Logger.Level != "debug" {
		t.Errorf("Invalid config %s", Get().Logger.Level)
	}
}

func TestSecondaryRegistry(t *testing.T) {
	Reset()

	// Mock command line arguments
	os.Args = append(os.Args, customConfig)

	type customConfiguration struct {
		Key string
	}

	type SecondaryRegistry struct {
		Custom customConfiguration
	}

	r2 := new(SecondaryRegistry)

	// This relies on default.dusk.toml
	if err := Load("custom", r2, nil); err != nil {
		t.Errorf("Failed parse: %v", err)
	}

	if Get().General.Network != "testnet" {
		t.Errorf("Invalid Network value: %s", Get().General.Network)
	}

	if r2.Custom.Key != "value" {
		t.Errorf("Invalid Custom Key value: %s", r2.Custom.Key)
	}

	if Get().Profile[0].Name != "heap" {
		t.Errorf("Invalid Profile [0]: %s", Get().Profile[0].Name)
	}

	if Get().Profile[1].Name != "cpu" {
		t.Errorf("Invalid Profile [1]: %s", Get().Profile[1].Name)
	}
}

func Reset() {
	pflag.CommandLine = &pflag.FlagSet{}
	pflag.Usage = func() {}

	viper.Reset()

	os.Args = initialArgs

	os.Unsetenv("DUSK_GENERAL_NETWORK")
	os.Unsetenv("DUSK_LOGGER_LEVEL")
}
