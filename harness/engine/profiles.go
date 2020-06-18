package engine

import (
	"strconv"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/spf13/viper"
)

// Profiles is a map with the node name as key and Profile function
type Profiles map[string]func(index int, node *DuskNode, walletPath string)

var profileList Profiles

// Profile1 builds the default dusk.toml definition
func Profile1(index int, node *DuskNode, walletPath string) {
	viper.Reset()
	viper.Set("general.network", "testnet")
	viper.Set("general.walletonly", "false")

	viper.Set("logger.output", node.Dir+"/dusk")
	viper.Set("gql.address", node.Cfg.Gql.Address)
	viper.Set("gql.network", node.Cfg.Gql.Network)
	viper.Set("gql.enabled", "true")

	viper.Set("rpc.network", node.Cfg.RPC.Network)
	if node.Cfg.RPC.Network == "unix" {
		viper.Set("rpc.address", node.Dir+node.Cfg.RPC.Address)
	} else {
		viper.Set("rpc.address", node.Cfg.RPC.Address)
	}
	viper.Set("rpc.sessionDurationMins", node.Cfg.RPC.SessionDurationMins)

	viper.Set("rpc.enabled", "true")
	viper.Set("rpc.rusk.network", node.Cfg.RPC.Rusk.Network)
	viper.Set("rpc.rusk.address", node.Cfg.RPC.Rusk.Address)
	viper.Set("rpc.rusk.contractTimeout", 6000)
	viper.Set("rpc.rusk.defaultTimeout", 1000)
	viper.Set("database.driver", heavy.DriverName)
	viper.Set("database.dir", node.Dir+"/chain/")
	viper.Set("wallet.store", node.Dir+"/walletDB/")
	viper.Set("wallet.file", walletPath+"/wallet-"+node.Id+".dat")
	viper.Set("network.seeder.addresses", []string{"127.0.0.1:8081"})
	viper.Set("network.port", strconv.Itoa(7600+index))
	viper.Set("mempool.maxSizeMB", "100")
	viper.Set("mempool.poolType", "hashmap")
	viper.Set("mempool.preallocTxs", "100")
	viper.Set("mempool.maxInvItems", "10000")
	viper.Set("genesis.legacy", true)

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
