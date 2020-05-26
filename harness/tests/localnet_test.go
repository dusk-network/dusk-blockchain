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
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/status"
)

var (
	localNetSizeStr = os.Getenv("NETWORK_SIZE")
	mockEnabled     = os.Getenv("MOCK_ENABLED")
	localNetSize    = 10
)

var localNet engine.Network
var workspace string

// TestMain sets up a temporarily local network of N nodes running from genesis block
// The network should be fully functioning and ready to accept messaging
func TestMain(m *testing.M) {
	flag.Parse()

	var err error
	workspace, err = ioutil.TempDir(os.TempDir(), "localnet-")
	if err != nil {
		log.Fatal(err)
	}

	if localNetSizeStr != "" {
		currentLocalNetSize, currentErr := strconv.Atoi(localNetSizeStr)
		if currentErr == nil {
			fmt.Println("Going to setup NETWORK_SIZE with custom value", "currentLocalNetSize", currentLocalNetSize)
			localNetSize = currentLocalNetSize
		}
	}

	// Create a network of N nodes
	for i := 0; i < localNetSize; i++ {
		node := engine.NewDuskNode(9500+i, 9000+i, "default")
		localNet.Nodes = append(localNet.Nodes, node)
	}

	if mockEnabled != "" {
		StartMockServer("127.0.0.1:8080")
	}

	var code int
	err = localNet.Bootstrap(workspace)

	if err == engine.ErrDisabledHarness {
		_ = os.RemoveAll(workspace)
		os.Exit(0)
	}

	if err != nil {
		// Failed temp network bootstrapping
		code = 1
		log.Fatal(err)

	} else {
		// Start all tests
		code = m.Run()
	}

	if *engine.KeepAlive != true {
		localNet.Teardown()
		_ = os.RemoveAll(workspace)
	}

	os.Exit(code)
}

// TestSendBidTransaction ensures that a valid bid transaction has been accepted
// by all nodes in the network within a particular time frame and within
// the same block
func TestSendBidTransaction(t *testing.T) {
	walletsPass := os.Getenv("DUSK_WALLET_PASS")

	t.Log("Send request to all nodes to loadWallet")
	for i := 0; i < localNetSize; i++ {
		_, err := localNet.LoadWalletCmd(uint(i), walletsPass)
		if err != nil {
			t.Fatal(err.Error())
		}
	}

	t.Log("Send request to node 0 to generate and process a Bid transaction")
	txidBytes, err := localNet.SendBidCmd(0, 10, 10)
	if err != nil {
		t.Fatal(err.Error())
	}

	txID := hex.EncodeToString(txidBytes)
	t.Logf("Bid transaction id: %s", txID)

	t.Log("Ensure all nodes have accepted this transaction at the same height")
	blockhash := ""
	for i := 0; i < len(localNet.Nodes); i++ {

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

	walletsPass := os.Getenv("DUSK_WALLET_PASS")

	t.Log("Send request to all nodes to loadWallet. This will start consensus")
	for i := 0; i < localNetSize; i++ {
		if _, err := localNet.LoadWalletCmd(uint(i), walletsPass); err != nil {
			st := status.Convert(err)
			if st.Message() != "wallet is already loaded" {
				// Better use of gRPC code 'st.Code()' later
				log.Fatal(err)
			}
		}
	}

	t.Log("Wait till we are at height 3")
	localNet.WaitUntil(t, 0, 3, 3*time.Minute, 5*time.Second)

	t.Log("Start a new node. This node falls behind during consensus")
	ind := localNetSize
	node := engine.NewDuskNode(9500+ind, 9000+ind, "default")
	localNet.Nodes = append(localNet.Nodes, node)

	if err := localNet.StartNode(ind, node, workspace); err != nil {
		t.Fatal(err.Error())
	}

	t.Log("Wait for two more blocks")
	localNet.WaitUntil(t, 0, 5, 2*time.Minute, 5*time.Second)

	t.Log("Ensure the new node has been synced up")
	localNet.WaitUntil(t, uint(ind), 5, 2*time.Minute, 5*time.Second)
}
