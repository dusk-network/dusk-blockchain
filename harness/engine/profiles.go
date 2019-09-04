package engine

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/spf13/viper"
	"strconv"
)

type Profiles map[string]func(index int, node *DuskNode, walletPath string)

var profileList Profiles

// Profile1 builds the default dusk.toml definition
func Profile1(index int, node *DuskNode, walletPath string) {
	viper.Reset()
	viper.Set("general.network", "testnet")
	viper.Set("logger.output", node.Dir+"/dusk")
	viper.Set("gql.port", node.Cfg.Gql.Port)
	viper.Set("gql.enabled", "true")
	viper.Set("rpc.port", node.Cfg.RPC.Port)
	viper.Set("rpc.enabled", "true")
	viper.Set("database.driver", heavy.DriverName)
	viper.Set("database.dir", node.Dir+"/chain/")
	viper.Set("wallet.store", node.Dir+"/walletDB/")
	viper.Set("wallet.file", walletPath+"/wallet-"+node.Id+".dat")
	viper.Set("network.seeder.addresses", []string{"127.0.0.1:8081"})
	viper.Set("network.port", strconv.Itoa(7000+index))
	viper.Set("mempool.maxSizeMB", "100")
	viper.Set("mempool.poolType", "hashmap")
	viper.Set("mempool.preallocTxs", "100")
	return
}

// Profile2 builds dusk.toml with lite driver enabled (suitable for bench testing)
func Profile2(index int, node *DuskNode, walletPath string) {

	Profile1(index, node, walletPath)
	viper.Set("database.driver", lite.DriverName)
}

func initProfiles() {
	profileList = make(Profiles, 0)
	profileList["default"] = Profile1
	profileList["defaultWithLite"] = Profile2
}
