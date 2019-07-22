package config

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config package should avoid importing any dusk-go packages in order to
// prevent any cyclic-dependancy issues

const (
	// current working dir
	searchPath1 = "."
	// home datadir
	searchPath2 = "$HOME/.dusk/"

	// name for the config file. Does not include extension.
	configFileName = "dusk"
)

var (
	r *Registry
)

// Registry stores all loaded configurations according to the config order
// NB It should be cheap to be copied by value
type Registry struct {
	UsedConfigFile string

	// All configuration groups
	General     generalConfiguration
	Database    databaseConfiguration
	Wallet      walletConfiguration
	Network     networkConfiguration
	Logger      loggerConfiguration
	Prof        profConfiguration
	RPC         rpcConfiguration
	Performance performanceConfiguration
	Mempool     mempoolConfiguration
}

// Load makes an attempt to read and unmershal any configs from flag, env and
// dusk config file.
//
// It  uses the following precedence order. Each item takes precedence over the item below it:
//  - flag
//  - env
//  - config
//  - key/value store (not used yet)
//  - default
//
// Dusk configuration file can be in form of TOML, JSON, YAML, HCL or Java
// properties config files
func Load() error {

	r = new(Registry)

	// Initialization
	if err := r.init(); err != nil {
		return err
	}

	// Validation and defaulting should be done by the consumers (packages) as
	// they will be the best at knowing what they expect

	return nil
}

// Get returns registry by value in order to avoid further modifications after
// initial configuration loading
func Get() Registry {
	return *r
}

func (r *Registry) init() error {

	// Make an attempt to find dusk.toml/dusk.json/dusk.yaml in any of the
	// provided paths below
	viper.SetConfigName(configFileName)

	// search paths
	viper.AddConfigPath(searchPath1)
	viper.AddConfigPath(searchPath2)

	// Initialize and parse flags
	confFile, err := loadFlags()

	if err != nil {
		return err
	}

	// confPath is overwritten by the one from command line
	if len(confFile) > 0 {
		viper.SetConfigFile(confFile)
	}

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("Error reading config file: %s", err)
	}

	defineENV()

	// Uncomment on debugging only. This will list all levels of configurations
	// viper.Debug()

	// Unmarshal all configurations from all conf levels to the registry struct
	if err := viper.Unmarshal(&r); err != nil {
		return fmt.Errorf("unable to decode into struct, %v", err)
	}

	r.UsedConfigFile = viper.ConfigFileUsed()

	return nil
}

func loadFlags() (string, error) {

	pflag.CommandLine.Init("Dusk node", pflag.ExitOnError)

	pflag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "Usage of %s:\n", "Dusk node")
		pflag.PrintDefaults()
	}

	// Define all supported flags.
	// All flags should be verified `loader_test.go/TestSupportedFlags`
	defineFlags()
	configFile := pflag.String("config", "", "Set path to the config file")

	// Bind all command line parameters to their corresponding file configs
	//
	// e.g CLI argument `--logger.level="warn"`` will overwrite the value from
	// `[logger] level = "info"`` in the loaded config file
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		return "", fmt.Errorf("unable bind pflags, %v", err)
	}

	pflag.Parse()

	return *configFile, nil
}

// define a set of flags as bindings to config file settings
// The settings that are needed to be passed frequently by CLI should be added here
func defineFlags() {
	_ = pflag.StringP("logger.level", "l", "", "override logger.level settings in config file")
	_ = pflag.StringP("general.network", "n", "testnet", "override general.network settings in config file")
	_ = pflag.StringP("network.port", "p", "7000", "port for the node to bind on")
	_ = pflag.StringP("logger.output", "o", "dusk", "specifies the log output")
	_ = pflag.StringP("database.dir", "b", "chain", "sets the blockchain database directory")
	_ = pflag.StringP("wallet.file", "w", "wallet.dat", "sets the wallet file to use")
	_ = pflag.StringP("wallet.store", "d", "walletDB", "sets the wallet database directory")
	_ = pflag.StringP("rpc.port", "r", "9000", "sets rpc server port")
}

// define a set of environment variables as bindings to config file settings
func defineENV() {

	// Bind config key general.network to ENV var DUSK_GENERAL_NETWORK
	if err := viper.BindEnv("general.network", "DUSK_GENERAL_NETWORK"); err != nil {
		fmt.Printf("defineENV %v", err)
	}

	if err := viper.BindEnv("logger.level", "DUSK_LOGGER_LEVEL"); err != nil {
		fmt.Printf("defineENV %v", err)
	}

	// DUSK_NETWORK_SEEDER_FIXED could be useful on setting up a local P2P
	// network by setting a single ENV with array of addresses
	//
	// export DUSK_NETWORK_SEEDER_FIXED="localhost:7000, localhost:7001, localhost:7002"
	if err := viper.BindEnv("network.seeder.fixed", "DUSK_NETWORK_SEEDER_FIXED"); err != nil {
		fmt.Printf("defineENV %v", err)
	}
}

// Mock should be used only in test packages. It could be useful when a unit
// test needs to be rerun with configs different from the default ones.
func Mock(m *Registry) {
	r = m
}

func init() {
	// By default Registry should be empty but not nil. In that way, consumers
	// (packages) can use their default values on unit testing
	r = new(Registry)
	r.Database.Driver = "lite_v0.1.0"
	r.General.Network = "testnet"
	r.Wallet.File = "wallet.dat"
	r.Wallet.Store = "walletDB"
}
