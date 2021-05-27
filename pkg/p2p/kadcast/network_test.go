// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcast_test

import (
	"bytes"
	"os"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

const (
	// basePort all listeners derive from.
	basePort = 10000
	// Number of the TestNetwork nodes.
	networkSize = 10
)

var enableProfiling = os.Getenv("CPU_PROFILE")

func kadcastBlock(b *block.Block, e *eventbus.EventBus) (*block.Block, error) {
	buf := new(bytes.Buffer)
	if err := message.MarshalBlock(buf, b); err != nil {
		return b, err
	}

	if err := topics.Prepend(buf, topics.Block); err != nil {
		return b, err
	}

	header := []byte{config.KadcastInitialHeight}
	m := message.NewWithHeader(topics.Block, *buf, header)

	e.Publish(topics.Kadcast, m)
	return b, nil
}

// TestBroadcastChunksMsg boostrap a kadcast network and make an attempt to
// broadcast a message to all network peers.
func TestBroadcastChunksMsg(t *testing.T) {
	// suppressing annoying INFO messages
	logrus.SetLevel(logrus.ErrorLevel)

	randBlocks := make([]*block.Block, networkSize)
	for i := 0; i < networkSize; i++ {
		randBlocks[i] = helper.RandomBlock(1, 3)
	}

	if enableProfiling == "1" {
		cwd, _ := os.Getwd()

		f, _ := os.Create(cwd + "/cpu.prof")
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Error("Could not start CPU profile: ", err)
		}

		defer func() {
			pprof.StopCPUProfile()
		}()
	}

	nodes, err := kadcast.TestNetwork(networkSize, basePort)
	if err != nil {
		t.Error(err)
	}

	// log.SetLevel(log.TraceLevel)
	for _, r := range nodes {
		kadcast.TraceRoutingState(r.Router)
	}

	time.Sleep(3 * time.Second)

	// Broadcast Chunk message. Each of the nodes makes an attempt to broadcast
	// a CHUNK message to the network
	/*
		If we assume constant transmission times, honest network partici-
		pants, and no packet loss in the underlying network, the propaga-
		tion method just discussed would result in an optimal broadcast
		tree. In this scenario, every node receives the block exactly once and
		hence no duplicate messages would be induced by this broadcast-
		ing operation.
	*/
	for i := 0; i < len(nodes); i++ {
		log.WithField("from_node", i).Infof("Broadcasting a message")

		// Publish topics.Kadcast with payload of a random block data to the
		// eventbus of this node. As a result, all of the network nodes should
		// have received the block only once as per beta value = 1
		blk, err := kadcastBlock(randBlocks[i], nodes[i].EventBus)
		if err != nil {
			t.Fatal(err)
		}

		kadcast.TestReceivedMsgOnce(t, nodes, i, blk)
	}
}
