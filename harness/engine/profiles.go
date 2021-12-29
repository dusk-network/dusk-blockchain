// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package engine

import (
	"log"
	"net"
	"strconv"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/spf13/viper"
)

// Profiles is a map with the node name as key and Profile function.
type Profiles map[string]func(index int, node *DuskNode, walletPath string)

var profileList Profiles

// Profile1 builds the default dusk.toml definition.
func Profile1(index int, node *DuskNode, walletPath string) {
	walletFileName := "wallet" + strconv.Itoa(index) + ".dat"

	viper.Reset()
	viper.Set("general.network", "devnet")
	viper.Set("general.walletonly", "false")
	viper.Set("general.safecallbacklistener", "false")
	viper.Set("general.testharness", "true")

	viper.Set("network.serviceflag", 1)
	viper.Set("network.maxconnections", 50)

	// blockgenerator.go
	viper.Set("timeout.timeoutgetlastcommittee", 5)
	// blockgenerator.go
	viper.Set("timeout.timeoutgetlastcertificate", 5)
	// blockgenerator.go
	viper.Set("timeout.timeoutgetmempooltxsbysize", 4)
	// initiator.go
	// sync.go
	viper.Set("timeout.timeoutgetlastblock", 5)
	// aggregator.go
	// candidatebroker.go
	// chain.go
	viper.Set("timeout.timeoutgetcandidate", 5)
	// chain.go
	viper.Set("timeout.timeoutclearwalletdatabase", 0)
	// aggregator.go
	viper.Set("timeout.timeoutverifycandidateblock", 5)
	// stakeautomaton.go
	viper.Set("timeout.timeoutsendstaketx", 5)
	// mempool.go datarequestor.go
	viper.Set("timeout.timeoutgetmempooltxs", 3)
	// roundresultsbroker.go
	viper.Set("timeout.timeoutgetroundresults", 5)
	// broker.go
	viper.Set("timeout.timeoutbrokergetcandidate", 2)
	// peer.go
	viper.Set("timeout.timeoutreadwrite", 60)
	// peer.go
	viper.Set("timeout.timeoutkeepalivetime", 30)
	viper.Set("timeout.timeoutdial", 5)

	viper.Set("logger.output", node.Dir+"/dusk")
	viper.Set("logger.level", "trace")
	viper.Set("logger.format", "json")

	viper.Set("gql.address", node.Cfg.Gql.Address)
	viper.Set("gql.network", node.Cfg.Gql.Network)
	viper.Set("gql.notification.brokersNum", "1")
	viper.Set("gql.notification.clientsPerBroker", "1000")
	viper.Set("gql.enabled", "true")

	viper.Set("rpc.network", node.Cfg.RPC.Network)

	if node.Cfg.RPC.Network == "unix" {
		viper.Set("rpc.address", node.Dir+node.Cfg.RPC.Address)
	} else {
		viper.Set("rpc.address", node.Cfg.RPC.Address)
	}

	viper.Set("rpc.sessionDurationMins", node.Cfg.RPC.SessionDurationMins)
	viper.Set("rpc.requireSession", node.Cfg.RPC.RequireSession)

	viper.Set("rpc.enabled", "true")

	addr := node.Cfg.RPC.Rusk.Address
	if node.Cfg.RPC.Rusk.Network == "unix" {
		addr = node.Dir + node.Cfg.RPC.Rusk.Address
	}

	viper.Set("rpc.rusk.network", node.Cfg.RPC.Rusk.Network)
	viper.Set("rpc.rusk.address", addr)
	viper.Set("rpc.rusk.contractTimeout", 20000)
	viper.Set("rpc.rusk.defaultTimeout", 1000)
	viper.Set("rpc.rusk.connectiontimeout", 10000)
	viper.Set("database.driver", heavy.DriverName)
	viper.Set("database.dir", node.Dir+"/chain/")
	viper.Set("wallet.store", node.Dir+"/walletDB/")
	viper.Set("wallet.file", walletPath+"/"+walletFileName)
	viper.Set("network.seeder.addresses", []string{"127.0.0.1:8081"})
	viper.Set("network.port", strconv.Itoa(7100+index))
	viper.Set("mempool.maxSizeMB", "100")
	viper.Set("mempool.poolType", "hashmap")
	viper.Set("mempool.preallocTxs", "100")
	viper.Set("mempool.maxInvItems", "10000")

	viper.Set("consensus.defaultlocktime", 1000)
	viper.Set("consensus.defaultoffset", 10)
	viper.Set("consensus.defaultamount", 50)
	viper.Set("consensus.consensustimeout", 5)
	viper.Set("consensus.usecompressedkeys", false)

	viper.Set("api.enabled", false)
	viper.Set("api.enabletls", false)
	viper.Set("api.address", "127.0.0.1:9199")
	viper.Set("api.expirationtime", 300)
}

// Profile2 builds dusk.toml with lite driver enabled (suitable for bench testing).
func Profile2(index int, node *DuskNode, walletPath string) {
	Profile1(index, node, walletPath)
	viper.Set("database.driver", lite.DriverName)
}

// Profile3 builds dusk.toml with kadcast enabled and gossip disabled.
func Profile3(index int, node *DuskNode, walletPath string) {
	Profile1(index, node, walletPath)

	basePortNumber := 10000
	laddr := getOutboundAddr(basePortNumber + index)

	bootstrappers := make([]string, 2)
	bootstrappers[0] = getOutboundAddr(basePortNumber)
	bootstrappers[1] = getOutboundAddr(basePortNumber + 1)

	viper.Set("kadcast.enabled", true)
	viper.Set("kadcast.address", laddr)
	viper.Set("kadcast.grpcport", 3*basePortNumber+index)
	viper.Set("kadcast.bootstrapAddr", bootstrappers)
}

func getOutboundAddr(port int) string {
	return "127.0.0.1:" + strconv.Itoa(port)
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
