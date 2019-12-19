package tests

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/harness/engine"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	localNetSize = 10
)

var localNet engine.Network

// TestMain sets up a temporarily local network of N nodes running from genesis block
// The network should be fully functioning and ready to accept messaging
func TestMain(m *testing.M) {

	flag.Parse()

	workspace, err := ioutil.TempDir(os.TempDir(), "localnet-")
	if err != nil {
		log.Fatal(err)
	}

	// Create a network of N nodes
	for i := 0; i < localNetSize; i++ {
		node := engine.NewDuskNode(
			strconv.Itoa(9500+i),
			strconv.Itoa(9000+i),
			"default",
		)
		localNet.Nodes = append(localNet.Nodes, node)
	}

	var code int
	err = localNet.Bootstrap(workspace)

	if err == engine.ErrDisabledHarness {
		os.RemoveAll(workspace)
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
		os.RemoveAll(workspace)
	}

	os.Exit(code)
}

// TestSendBidTransaction ensures that a valid bid transaction has been accepted
// by all nodes in the network within a particular time frame and within
// the same block
func TestSendBidTransaction(t *testing.T) {

	walletsPass := os.Getenv("DUSK_WALLET_PASS")
	if len(walletsPass) == 0 {
		t.Logf("Empty DUSK_WALLET_PASS")
	}

	// Send request to all nodes to loadWallet
	for i := 0; i < localNetSize; i++ {
		_, err := localNet.SendCommand(uint(i), "loadwallet", []string{walletsPass})
		if err != nil {
			t.Fatal(err.Error())
		}
	}

	// Send request to node 0 to generate and process a Bid transaction
	data, err := localNet.SendCommand(0, "bid", []string{"10", "10"})
	if err != nil {
		t.Fatal(err.Error())
	}

	var resp string
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatal(err.Error())
	}

	txid := strings.Trim(resp, "Success! TxID: ")
	t.Logf("Bid transaction id: %s", txid)

	// Ensure all nodes have accepted this transaction at the same height
	blockhash := ""
	for i, node := range localNet.Nodes {
		condition := func() bool {
			// Construct query to fetch txid
			query := fmt.Sprintf(
				"{\"query\" : \"{ transactions (txid: \\\"%s\\\") { txid blockhash } }\"}",
				txid)

			var resp map[string]map[string][]map[string]string
			err := localNet.SendQuery(uint(i), query, &resp)
			if err != nil {
				return false
			}

			result, ok := resp["data"]
			if !ok {
				return false
			}

			if len(result["transactions"]) == 0 {
				// graphql request processed but still txid not found
				return false
			}

			if i == 0 {
				blockhash = result["transactions"][0]["blockhash"]
			}

			if len(blockhash) != 0 && blockhash != result["transactions"][0]["blockhash"] {
				// the case where the network has inconsistency and same tx has been
				// accepted within different blocks
				return false
			}

			if result["transactions"][0]["txid"] != txid {
				return false
			}

			return true
		}

		// asserts that given condition will be met in 1 minute, by checking condition function each second.
		if !assert.Eventuallyf(t, condition, 1*time.Minute, time.Second, "failed node %s", node.Id) {
			break
		}
	}
}

// TestCatchup tests that a node which falls behind during consensus
// will properly catch up and re-join the consensus execution trace.
func TestCatchup(t *testing.T) {
	walletsPass := os.Getenv("DUSK_WALLET_PASS")
	if len(walletsPass) == 0 {
		t.Logf("Empty DUSK_WALLET_PASS")
	}

	// Send request to all nodes to loadWallet
	for i := 0; i < localNetSize; i++ {
		_, err := localNet.SendCommand(uint(i), "loadwallet", []string{walletsPass})
		if err != nil {
			t.Fatal(err.Error())
		}
	}

	waitHeight := 6
	waitUntil := func() bool {
		// Construct query to fetch block height
		query := fmt.Sprintf(
			"{\"query\" : \"{ blocks (last: 1) { header { height } } }\"}",
		)

		var result map[string]map[string][]map[string]map[string]int
		if err := localNet.SendQuery(1, query, &result); err != nil {
			return false
		}

		if result["data"]["blocks"][0]["header"]["height"] >= waitHeight {
			return true
		}

		return false
	}

	// Wait till we are at height 6
	assert.Eventually(t, waitUntil, 200*time.Second, 5*time.Second)

	// Stop consensus for node 0
	if _, err := localNet.SendCommand(0, "publishTopic", []string{"stopconsensus", ""}); err != nil {
		t.Fatal(err)
	}

	// Wait for two more blocks
	waitHeight += 2
	assert.Eventually(t, waitUntil, 100*time.Second, 5*time.Second)

	// Ensure node syncs up
	ensureSynced := func() bool {
		// Construct query to fetch block height
		query := fmt.Sprintf(
			"{\"query\" : \"{ blocks (last: 1) { header { height } } }\"}",
		)

		var result map[string]map[string][]map[string]map[string]int
		if err := localNet.SendQuery(0, query, &result); err != nil {
			return false
		}

		heightNode0 := result["data"]["blocks"][0]["header"]["height"]
		var result2 map[string]map[string][]map[string]map[string]int
		if err := localNet.SendQuery(1, query, &result2); err != nil {
			return false
		}

		heightNode1 := result2["data"]["blocks"][0]["header"]["height"]

		return heightNode0 == heightNode1
	}

	assert.Eventually(t, ensureSynced, 200*time.Second, 1*time.Second)
}
