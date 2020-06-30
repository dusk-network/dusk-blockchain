package engine

import (
	"log"
	"net"
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

	viper.Set("logger.output", node.Dir+"/dusk")
	viper.Set("logger.level", "warn")

	viper.Set("gql.address", node.Cfg.Gql.Address)
	viper.Set("gql.network", node.Cfg.Gql.Network)
	viper.Set("gql.enabled", "true")

	viper.Set("rpc.network", node.Cfg.RPC.Network)
	if node.Cfg.RPC.Network == "unix" {
		viper.Set("rpc.address", node.Dir+node.Cfg.RPC.Address)
	} else {
		viper.Set("rpc.address", node.Cfg.RPC.Address)
	}

	viper.Set("rpc.enabled", "true")
	viper.Set("database.driver", heavy.DriverName)
	viper.Set("database.dir", node.Dir+"/chain/")
	viper.Set("wallet.store", node.Dir+"/walletDB/")
	viper.Set("wallet.file", walletPath+"/wallet-"+node.Id+".dat")
	viper.Set("network.seeder.addresses", []string{"127.0.0.1:8081"})
	viper.Set("network.port", strconv.Itoa(7100+index))
	viper.Set("mempool.maxSizeMB", "100")
	viper.Set("mempool.poolType", "hashmap")
	viper.Set("mempool.preallocTxs", "100")
	viper.Set("mempool.maxInvItems", "10000")
	viper.Set("consensus.defaultlocktime", 1000)
	viper.Set("consensus.defaultoffset", 10)
	viper.Set("consensus.defaultamount", 50)
}

// Profile2 builds dusk.toml with lite driver enabled (suitable for bench testing)
func Profile2(index int, node *DuskNode, walletPath string) {

	Profile1(index, node, walletPath)
	viper.Set("database.driver", lite.DriverName)
}

// Profile3 builds dusk.toml with kadcast enabled and gossip disabled
func Profile3(index int, node *DuskNode, walletPath string) {

	Profile1(index, node, walletPath)

	viper.Set("kadcast.enabled", true)
	basePortNumber := 10000

	laddr := getOutboundAddr(basePortNumber + index)
	viper.Set("kadcast.address", laddr)
	viper.Set("kadcast.maxDelegatesNum", 1)
	viper.Set("kadcast.raptor", true)

	bootstrappers := make([]string, 4)
	bootstrappers[0] = getOutboundAddr(basePortNumber)
	bootstrappers[1] = getOutboundAddr(basePortNumber + 1)
	bootstrappers[2] = getOutboundAddr(basePortNumber + 2)
	bootstrappers[3] = getOutboundAddr(basePortNumber + 3)
	viper.Set("kadcast.bootstrappers", bootstrappers)
}

func getOutboundAddr(port int) string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = conn.Close()
	}()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String() + ":" + strconv.Itoa(port)
}

func initProfiles() {
	profileList = make(Profiles)
	profileList["default"] = Profile1
	profileList["defaultWithLite"] = Profile2
	profileList["kadcast"] = Profile3
}
