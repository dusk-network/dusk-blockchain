package tests

import (
	"fmt"
	"github.com/dusk-network/dusk-blockchain/harness/engine"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	localNetSize = 4
)

var localNet engine.Network

// TestMain sets up a temporarily local network of N nodes running from genesis block
// The network should be fully functioning and ready to accept messaging
func TestMain(m *testing.M) {

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

	localNet.Bootstrap(workspace)

	// Start all tests
	code := m.Run()

	localNet.Teardown()
	os.RemoveAll(workspace)

	os.Exit(code)
}

// TestSendBidTransaction ensures that a valid bid transaction has been accepted
// by all nodes in the network within a particular time frame and within
// the same block
func TestSendBidTransaction(t *testing.T) {

	// Send request to node 0 to generate and process a Bid transaction
	data, err := localNet.SendCommand(0, "sendBidTx", []string{"10", "1", os.Getenv("DUSK_WALLET_PASS")})
	if err != nil {
		t.Error(err.Error())
	}

	txid := string(data)
	t.Logf("Bid transaction id: %s", txid)
	txid = strings.Replace(txid, "\"", "", -1)

	// Ensure all nodes have accepted this transaction at the same height
	blockhash := ""
	for i, node := range localNet.Nodes {
		condition := func() bool {
			// Construct query to fetch txid
			query := fmt.Sprintf("{ transactions (txid: \"%s\") { txid blockhash } }", txid)
			result, err := localNet.SendQuery(uint(i), query)
			if err != nil {
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

		// asserts that given condition will be met in 120 seconds, by checking condition function each second.
		if !assert.Eventuallyf(t, condition, 120*time.Second, time.Second, "failed node %s", node.Id) {
			break
		}
	}
}
