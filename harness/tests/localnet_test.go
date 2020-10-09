package tests

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/harness/engine"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var (
	localNetSizeStr = os.Getenv("NETWORK_SIZE")
	localNetSize    = 10
)

var localNet engine.Network
var workspace string

// TestMain sets up a temporarily local network of N nodes running from genesis block
// The network should be fully functioning and ready to accept messaging
func TestMain(m *testing.M) {
	var err error
	flag.Parse()

	// create the temp-dir workspace. Quit on error
	workspace, err = ioutil.TempDir(os.TempDir(), "localnet-")
	if err != nil {
		quit(localNet, workspace, err)
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

	fmt.Println("GRPC Session enabled?", localNet.IsSessionRequired())

	// Create a network of N nodes
	for i := 0; i < localNetSize; i++ {
		node := engine.NewDuskNode(9500+i, 9000+i, "default", localNet.IsSessionRequired())
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

// TestSendBidTransaction ensures that a valid bid transaction has been accepted
// by all nodes in the network within a particular time frame and within
// the same block
func TestSendBidTransaction(t *testing.T) {

	localNet.LoadNetworkWallets(t, localNet.Size())

	t.Log("Send request to node 0 to generate and process a Bid transaction")
	txidBytes, err := localNet.SendBidCmd(0, 10, 10)
	if err != nil {
		t.Fatal(err)
	}

	for i := 1; i < localNet.Size(); i++ {
		t.Logf("Send request to node %d to generate and process a Bid transaction", i)
		if _, err := localNet.SendBidCmd(uint(i), 10, 10); err != nil {
			t.Error(err)
		}
	}

	txID := hex.EncodeToString(txidBytes)
	t.Logf("Bid transaction id: %s", txID)

	t.Log("Ensure all nodes have accepted this transaction at the same height")
	blockhash := ""
	for i := 0; i < localNet.Size(); i++ {

		bh := localNet.WaitUntilTx(t, uint(i), txID)

		if len(bh) == 0 {
			t.Fatal("empty blockhash")
		}

		if len(blockhash) != 0 && blockhash != bh {
			// the case where the network has inconsistency and same tx has been
			// accepted within different blocks
			t.Fatal("same tx hash has been accepted within different blocks")
		}

		if i == 0 {
			blockhash = bh
		}
	}
}

// TestCatchup tests that a node which falls behind during consensus
// will properly catch up and re-join the consensus execution trace.
func TestCatchup(t *testing.T) {

	localNet.LoadNetworkWallets(t, localNet.Size())

	t.Log("Wait till we are at height 3")
	localNet.WaitUntil(t, 0, 3, 3*time.Minute, 5*time.Second)

	t.Log("Start a new node. This node falls behind during consensus")
	ind := localNetSize
	node := engine.NewDuskNode(9500+ind, 9000+ind, "default", localNet.IsSessionRequired())
	localNet.AddNode(node)

	if err := localNet.StartNode(ind, node, workspace); err != nil {
		t.Fatal(err.Error())
	}

	t.Log("Wait for two more blocks")
	localNet.WaitUntil(t, 0, 5, 2*time.Minute, 5*time.Second)

	t.Log("Ensure the new node has been synced up")
	localNet.WaitUntil(t, uint(ind), 5, 2*time.Minute, 5*time.Second)
}

func TestSendStakeTransaction(t *testing.T) {

	localNet.LoadNetworkWallets(t, localNet.Size())

	t.Log("Send request to node 1 to generate and process a Bid transaction")
	txidBytes, err := localNet.SendStakeCmd(1, 10, 10)
	if err != nil {
		t.Fatal(err)
	}

	txID := hex.EncodeToString(txidBytes)
	t.Logf("Stake transaction id: %s", txID)

	t.Log("Ensure all nodes have accepted stake transaction at the same height")
	blockhash := ""
	for i := 0; i < localNet.Size(); i++ {

		bh := localNet.WaitUntilTx(t, uint(i), txID)

		if len(bh) == 0 {
			t.Fatal("empty blockhash")
		}

		if len(blockhash) != 0 && blockhash != bh {
			// the case where the network has inconsistency and same tx has been
			// accepted within different blocks
			t.Fatal("same tx hash has been accepted within different blocks")
		}

		if i == 0 {
			blockhash = bh
		}
	}
}

// TestMultipleBiddersProvisioners should be helpful on long-run testing. It
// should ensure that in a network of multiple Bidders and Provisioners
// consensus is stable and consistent
func TestMultipleBiddersProvisioners(t *testing.T) {

	if !*engine.KeepAlive {
		// Runnable in keepalive mode only
		t.SkipNow()
	}

	logrus.Infof("TestMultipleBiddersProvisioners with staking")

	// Network setup procedure
	localNet.LoadNetworkWallets(t, localNet.Size())

	defaultLocktime := uint64(100000)

	// Load Bidders and Provisioners
	for i := 0; i < localNet.Size(); i++ {
		time.Sleep(100 * time.Millisecond)
		if i%2 == 0 {
			t.Logf("Send request to node %d to generate and process a Bid transaction", i)
			if _, err := localNet.SendBidCmd(uint(i), 10, defaultLocktime); err != nil {
				t.Error(err)
			}
		} else {
			t.Logf("Send request to node %d to generate and process a Stake transaction", i)
			if _, err := localNet.SendStakeCmd(uint(i), 10, defaultLocktime); err != nil {
				t.Error(err)
			}
		}
	}

	deployNewNode := func() {
		ind := localNetSize
		node := engine.NewDuskNode(9500+ind, 9000+ind, "default", localNet.IsSessionRequired())
		localNet.AddNode(node)

		if err := localNet.StartNode(ind, node, workspace); err != nil {
			fmt.Print("failed starting a node")
		}
	}

	// Deploy new node an hour after bootstrapping the network
	// This allows us to monitor re-sync process with more than 500 blocks difference
	time.AfterFunc(1*time.Hour, deployNewNode)
}
