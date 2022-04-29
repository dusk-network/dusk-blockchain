// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package tests

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/harness/engine"
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/sirupsen/logrus"
)

var (
	localNetSizeStr = os.Getenv("DUSK_NETWORK_SIZE")
	localNetSize    = 10

	// tomlProfile could be 'default', 'kadcast', kadcast_uds.
	tomlProfile = os.Getenv("DUSK_NETWORK_PROFILE")

	// networkReuse is flag to instruct test-harness to reuse the existing state.
	_, networkReuse = os.LookupEnv("DUSK_NETWORK_REUSE")
)

var (
	localNet  engine.Network
	workspace string
)

// TestMain sets up a temporarily local network of N nodes running from genesis
// block.
//
// The network should be fully functioning and ready to accept messaging.
func TestMain(m *testing.M) {
	flag.Parse()

	localNet.Reuse = networkReuse

	// create the temp-dir workspace. Quit on error
	workspace = path.Join(os.TempDir(), "localnet")

	if !localNet.Reuse {
		if err := os.Mkdir(workspace, 0700); err != nil {
			fmt.Println("Cleaning temp directory", workspace)

			if err := os.RemoveAll(workspace); err == nil {
				os.Mkdir(workspace, 0700)
			}
		}
	} else {
		fmt.Println("Network reuse mode enabled")
	}

	// set the network size
	if localNetSizeStr != "" {
		currentLocalNetSize, currentErr := strconv.Atoi(localNetSizeStr)
		if currentErr != nil {
			quit(localNet, workspace, currentErr)
		}

		fmt.Println("Going to setup NETWORK_SIZE with custom value", "currentLocalNetSize", currentLocalNetSize)
		localNetSize = currentLocalNetSize
	}

	// Select network type based on nodes profile name
	switch tomlProfile {
	case "", "default":
		{
			tomlProfile = "default"
			localNet.NetworkType = engine.GossipNetwork
		}
	case "kadcast":
		fallthrough
	case "kadcast_uds":
		{
			localNet.NetworkType = engine.KadcastNetwork
		}
	}

	fmt.Println("GRPC Session enabled?", localNet.IsSessionRequired())

	// Create a network of N nodes
	for i := 0; i < localNetSize; i++ {
		node := engine.NewDuskNode(9500+i, 9000+i, tomlProfile, localNet.IsSessionRequired())
		localNet.AddNode(node)
	}

	if err := localNet.Bootstrap(workspace); err != nil {
		fmt.Println(err)

		if err == engine.ErrDisabledHarness {
			_ = os.RemoveAll(workspace)
			os.Exit(0)
		}

		// Failed temp network bootstrapping
		log.Fatal(err)
		exit(localNet, workspace, 1)
	}

	fmt.Println("Nodes started")

	// Start all tests
	code := m.Run()

	if *engine.KeepAlive {
		monitorNetwork()
	}

	// finalize the tests and exit
	exit(localNet, workspace, code)
}

func quit(network engine.Network, workspace string, err error) {
	if err != nil {
		log.Fatal(err)
		exit(network, workspace, 1)
	}

	exit(network, workspace, 0)
}

func exit(localNet engine.Network, workspace string, code int) {
	if *engine.KeepAlive != true {
		localNet.Teardown()

		_ = os.RemoveAll(workspace)
	}

	os.Exit(code)
}

func monitorNetwork() {
	logrus.Info("Network status monitoring")

	// Monitor network, consensus, enviornement status  etc
	// Pending to add more run-time assertion points
	for {
		// Monitor network tip
		networkTip, err := localNet.IsSynced(5)
		if err != nil {
			// Network is not synced completely
			logrus.Error(err)
		}

		logrus.WithField("tip", networkTip).Info("Network status")

		// Check if race conditions reported for any of the network nodes
		// TODO:

		// Monitoring iteration delay
		time.Sleep(10 * time.Second)
	}
}

// TestCatchup tests that a node which falls behind during consensus will
// properly catch up and re-join the consensus execution trace.
func TestCatchup(t *testing.T) {
	t.Log("Wait till we are at height 3")

	localNet.WaitUntil(t, 0, 3, 3*time.Minute, 5*time.Second)

	t.Log("Start a new node. This node falls behind during consensus")

	ind := localNetSize

	node := engine.NewDuskNode(9500+ind, 9000+ind, tomlProfile, localNet.IsSessionRequired())
	localNet.AddNode(node)

	if err := localNet.StartNode(ind, node, workspace); err != nil {
		t.Fatal(err.Error())
	}

	t.Log("Wait for two more blocks")
	localNet.WaitUntil(t, 0, 5, 2*time.Minute, 5*time.Second)

	t.Log("Ensure the new node has been synced up")
	localNet.WaitUntil(t, uint(ind), 5, 2*time.Minute, 5*time.Second)
}

// TestMultipleProvisioners should be helpful on long-run testing. It should
// ensure that in a network of multiple Provisioners with equal stake, consensus
// is stable and consistent.
func TestMultipleProvisioners(t *testing.T) {
	if !*engine.KeepAlive {
		// Runnable in keepalive mode only
		t.SkipNow()
	}

	localNet.PrintWalletsInfo(t)
	logrus.Infof("TestMultipleProvisioners with staking")

	defaultLocktime := uint64(100000)

	amount := config.DUSK * 10000

	// Send a bunch of Stake transactions
	// All nodes are Provisioners
	for i := uint(0); i < uint(localNet.Size()); i++ {
		time.Sleep(100 * time.Millisecond)
		t.Logf("Node %d sending a Stake transaction", i)

		if _, err := localNet.SendStakeCmd(i, amount, defaultLocktime); err != nil {
			t.Log(err.Error())
		}
	}

	deployNewNode := func() {
		ind := localNetSize
		node := engine.NewDuskNode(9500+ind, 9000+ind, tomlProfile, localNet.IsSessionRequired())
		localNet.AddNode(node)

		if err := localNet.StartNode(ind, node, workspace); err != nil {
			fmt.Print("failed starting a node")
		}
	}

	// Deploy new node an hour after bootstrapping the network
	// This allows us to monitor re-sync process with more than 500 blocks difference
	time.AfterFunc(1*time.Hour, deployNewNode)
}

// TestMeasureNetworkTPS is a placeholder for a simple test definition on top of
// which network TPS metric should be collected.
func TestMeasureNetworkTPS(t *testing.T) {
	if !*engine.KeepAlive {
		// Runnable in keepalive mode only
		t.SkipNow()
	}

	// Start monitoring TPS metric of the network
	localNet.MonitorTPS(4 * time.Second)
}

// TestForgedBlock propagates a forged block with height >100000 to the network.
// It should ensure that this will not impact negatively consensus.
func TestForgedBlock(t *testing.T) {
	blk := helper.RandomBlock(1000000, 3)

	buf := new(bytes.Buffer)
	if err := message.MarshalBlock(buf, blk); err != nil {
		t.Fatal(err)
	}

	if err := topics.Prepend(buf, topics.Block); err != nil {
		t.Fatal(err)
	}

	networkTip, _ := localNet.GetLastBlockHeight(0)

	for i := uint(0); i < uint(localNet.Size()); i++ {
		time.Sleep(100 * time.Millisecond)
		t.Logf("Sending a forged block to Node %d", i)

		if err := localNet.SendWireMsg(i, buf.Bytes(), 2); err != nil {
			t.Log(err)
		}
	}

	// Waits for at least 5 consecutive blocks to ensure consensus has not been stalled
	localNet.WaitUntil(t, 0, networkTip+5, 3*time.Minute, 5*time.Second)
}
