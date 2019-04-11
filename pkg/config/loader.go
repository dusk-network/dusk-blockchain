package config

import (
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
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

var parser *Parser

type Parser struct {
	UsedConfigFile string

	// All configuration groups
	General  GeneralConfiguration
	Database DatabaseConfiguration
	Network  NetworkConfiguration
	Logger   LoggerConfiguration
	Profile  ProfileConfiguration
}

// Parse makes an attempt to read and unmershal any configs from flag, env and
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
func Parse() error {

	parser = new(Parser)

	// Initialization
	if err := parser.init(); err != nil {
		return err
	}

	// Validation and defaulting should be done by the consumers (packages) as
	// they will be the best at knowing what they expect

	return nil
}

// Get returns parser by value in order to avoid further modifications after
// initial configuration loading
func Get() Parser {
	return *parser
}

func (parser *Parser) init() error {

	// Make an attempt to find dusk.toml/dusk.json/dusk.yaml in any of the
	// provided paths below
	viper.SetConfigName(configFileName)

	// search paths
	viper.AddConfigPath(searchPath1)
	viper.AddConfigPath(searchPath2)

	// Initialize and parse flags
	confFile, err := parser.loadFlags()

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

	parser.defineENV()

	// Uncomment on debugging only. This will list all levels of configurations
	// viper.Debug()

	// Unmarshal all configurations from all conf levels to the parser struct
	if err := viper.Unmarshal(&parser); err != nil {
		return fmt.Errorf("unable to decode into struct, %v", err)
	}

	parser.UsedConfigFile = viper.ConfigFileUsed()

	return nil
}

func (parser *Parser) loadFlags() (string, error) {

	pflag.CommandLine.Init("Dusk node", pflag.ExitOnError)

	pflag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "Usage of %s:\n", "Dusk node")
		pflag.PrintDefaults()
	}

	// Define all supported flags.
	// All flags should be verified `loader_test.go/TestSupportedFlags`
	parser.defineFlags()
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
func (parser *Parser) defineFlags() {
	_ = pflag.String("logger.level", "", "override logger.level settings in config file")
	_ = pflag.String("general.network", "testnet", "override general.network settings in config file")
}

// define a set of environment variables as bindings to config file settings
func (parser *Parser) defineENV() {

	// Bind config key general.network to ENV var DUSK_GENERAL_NETWORK
	if err := viper.BindEnv("general.network", "DUSK_GENERAL_NETWORK"); err != nil {
		fmt.Printf("defineENV %v", err)
	}

	if err := viper.BindEnv("logger.level", "DUSK_LOGGER_LEVEL"); err != nil {
		fmt.Printf("defineENV %v", err)
	}
}
