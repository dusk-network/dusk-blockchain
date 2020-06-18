package main

import (
	"fmt"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

//## SBA Consensus Tests

var (
	CONSENSUS_TEST_MODE = os.Getenv("CONSENSUS_TEST_MODE")
)

type testNode struct {
	isRunning      bool
	isInitiated    bool
	wasStopped     bool
	privateKey     *interface{}
	address        string
	port           int
	url            string
	listener       net.Listener
	node           *interface{}
	enode          *interface{}
	service        *interface{}
	transactions   map[byte]struct{}
	transactionsMu sync.Mutex
	blocks         map[uint64]block.Block
	lastBlock      uint64
	txsSendCount   *int64
	txsChainCount  map[uint64]int64
}

type hook func(block *block.Block, validator *testNode, tCase *testCase, currentTime time.Time) error

type testCase struct {
	name                 string
	isSkipped            bool
	numPeers             int
	numBlocks            int
	txPerPeer            int
	maliciousPeers       map[int]func() interface{}
	networkRates         map[int]interface{}
	beforeHooks          map[int]hook
	afterHooks           map[int]hook
	sendTransactionHooks map[int]func(service *interface{}, key *interface{}, fromAddr []byte, toAddr []byte) (*transactions.Transaction, error)
	stopTime             map[int]time.Time
	genesisHook          func(g *interface{}) *interface{}
	mu                   sync.RWMutex
}

func TestTendermintSuccess(t *testing.T) {
	if testing.Short() || CONSENSUS_TEST_MODE != "sba" {
		t.Skip("skipping consensus test")
	}

	cases := []*testCase{
		{
			name:      "no malicious",
			numPeers:  5,
			numBlocks: 15,
			txPerPeer: 2,
		},
	}

	for _, testCase := range cases {
		testCase := testCase
		t.Run(fmt.Sprintf("test case %s", testCase.name), func(t *testing.T) {
			runTest(t, testCase)
		})
	}
}

func runTest(t *testing.T, test *testCase) {
	if test.isSkipped {
		t.SkipNow()
	}
}
