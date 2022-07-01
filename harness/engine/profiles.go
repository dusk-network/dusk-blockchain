// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package engine

import (
	"fmt"
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
func Profile1(index int, node *DuskNode, consensusKeysPath string) {
	consensusKeysFileName := "node_" + strconv.Itoa(index) + ".keys"

	viper.Reset()
	viper.Set("general.network", "testnet")
	viper.Set("general.testharness", "true")

	viper.Set("network.serviceflag", 1)

	viper.Set("logger.output", node.Dir+"/dusk")
	viper.Set("logger.level", "debug")
	viper.Set("logger.format", "json")

	viper.Set("gql.address", node.Cfg.Gql.Address)
	viper.Set("gql.network", node.Cfg.Gql.Network)
	viper.Set("gql.notification.brokersNum", "1")
	viper.Set("gql.notification.clientsPerBroker", "1000")
	viper.Set("gql.enabled", "true")

	addr := node.Cfg.RPC.Rusk.Address
	if node.Cfg.RPC.Rusk.Network == "unix" {
		addr = node.Dir + node.Cfg.RPC.Rusk.Address
	}

	viper.Set("rpc.rusk.network", node.Cfg.RPC.Rusk.Network)
	viper.Set("rpc.rusk.address", addr)
	viper.Set("rpc.rusk.contractTimeout", 20000)
	viper.Set("rpc.rusk.defaultTimeout", 1000)
	viper.Set("rpc.rusk.connectiontimeout", 10000)

	// database configs
	viper.Set("database.driver", heavy.DriverName)
	viper.Set("database.dir", node.Dir+"/chain/")

	// mempool configs
	viper.Set("mempool.maxSizeMB", "100")
	viper.Set("mempool.updates.numNodes", "1")
	// viper.Set("mempool.poolType", "hashmap")
	// viper.Set("mempool.preallocTxs", "100")
	viper.Set("mempool.poolType", "diskpool")
	viper.Set("mempool.diskpoolDir", node.Dir+"/mempool.db")

	viper.Set("mempool.maxInvItems", "10000")
	viper.Set("mempool.propagateTimeout", "100ms")
	viper.Set("mempool.propagateBurst", 1)

	// consensus config
	viper.Set("consensus.consensustimeout", 5)
	viper.Set("consensus.usecompressedkeys", false)
	viper.Set("consensus.keysfile", consensusKeysPath+"/"+consensusKeysFileName)

	viper.Set("state.persistevery", 100)

	// internal rpc timeouts
	viper.Set("timeout.timeoutgetmempooltxsbysize", 4)
	viper.Set("timeout.timeoutgetmempooltxs", 3)
}

// Profile2 builds dusk.toml with lite driver enabled (suitable for bench testing).
func Profile2(index int, node *DuskNode, walletPath string) {
	Profile1(index, node, walletPath)
	viper.Set("database.driver", lite.DriverName)
}

// Profile3 builds dusk.toml with kadcast enabled and gossip disabled.
func Profile3(index int, node *DuskNode, walletPath string) {
	Profile1(index, node, walletPath)

	const (
		basePortNumber = 10000
		baseAddr       = "127.0.0.1"
	)

	bootstrappers := make([]string, 2)
	bootstrappers[0] = fmt.Sprintf("%s:%d", baseAddr, basePortNumber)
	bootstrappers[1] = fmt.Sprintf("%s:%d", baseAddr, basePortNumber+1)

	viper.Set("kadcast.enabled", true)
	viper.Set("kadcast.address", fmt.Sprintf("%s:%d", baseAddr, basePortNumber+index))
	viper.Set("kadcast.bootstrapAddr", bootstrappers)

	viper.Set("kadcast.grpc.Network", "tcp")
	viper.Set("kadcast.grpc.Address", fmt.Sprintf("%s:%d", baseAddr, 3*basePortNumber+index))
	viper.Set("kadcast.grpc.DialTimeout", 10)
}

// Profile4 builds dusk.toml with kadcast enabled over unix socket.
func Profile4(index int, node *DuskNode, walletPath string) {
	Profile1(index, node, walletPath)

	const (
		basePortNumber = 10000
		baseAddr       = "127.0.0.1"
	)

	bootstrappers := make([]string, 2)
	bootstrappers[0] = fmt.Sprintf("%s:%d", baseAddr, basePortNumber)
	bootstrappers[1] = fmt.Sprintf("%s:%d", baseAddr, basePortNumber+1)

	viper.Set("kadcast.enabled", true)
	viper.Set("kadcast.address", fmt.Sprintf("%s:%d", baseAddr, basePortNumber+index))
	viper.Set("kadcast.bootstrapAddr", bootstrappers)

	viper.Set("kadcast.grpc.Network", "unix")
	viper.Set("kadcast.grpc.Address", node.Dir+"/rusk-grpc.sock")
	viper.Set("kadcast.grpc.DialTimeout", 10)
}

//nolint
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
	profileList["kadcast_uds"] = Profile4
}
