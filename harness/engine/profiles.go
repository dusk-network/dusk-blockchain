package engine

import (
	"strconv"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/spf13/viper"
)

type Profiles map[string]func(index int, node *DuskNode, walletPath string)

var profileList Profiles

// Profile1 builds the default dusk.toml definition
func Profile1(index int, node *DuskNode, walletPath string) {
	viper.Reset()
	viper.Set("general.network", "testnet")
	viper.Set("logger.output", node.Dir+"/dusk")
	viper.Set("gql.address", node.Cfg.Gql.Address)
	viper.Set("gql.enabled", "true")
	viper.Set("rpc.network", node.Cfg.RPC.Network)
	viper.Set("rpc.address", node.Cfg.RPC.Address)
	viper.Set("rpc.enabled", "true")
	viper.Set("rpc.user", "default")
	viper.Set("rpc.pass", "default")
	viper.Set("database.driver", heavy.DriverName)
	viper.Set("database.dir", node.Dir+"/chain/")
	viper.Set("wallet.store", node.Dir+"/walletDB/")
	viper.Set("wallet.file", walletPath+"/wallet-"+node.Id+".dat")
	viper.Set("network.seeder.addresses", []string{"127.0.0.1:8081"})
	viper.Set("network.port", strconv.Itoa(7100+index))
	viper.Set("mempool.maxSizeMB", "100")
	viper.Set("mempool.poolType", "hashmap")
	viper.Set("mempool.preallocTxs", "100")
	viper.Set("consensus.defaultlocktime", 1000)
	viper.Set("consensus.defaultoffset", 10)
	viper.Set("consensus.defaultamount", 50)
}

// Profile2 builds dusk.toml with lite driver enabled (suitable for bench testing)
func Profile2(index int, node *DuskNode, walletPath string) {

	Profile1(index, node, walletPath)
	viper.Set("database.driver", lite.DriverName)
}

func initProfiles() {
	profileList = make(Profiles)
	profileList["default"] = Profile1
	profileList["defaultWithLite"] = Profile2
}
