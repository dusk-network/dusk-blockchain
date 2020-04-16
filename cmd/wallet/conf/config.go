package conf

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

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

func InitConfig() Registry {
	viper.SetConfigName("dusk")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.dusk/")

	if err := viper.ReadInConfig(); err != nil {
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
