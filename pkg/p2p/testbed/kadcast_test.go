// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

//go:build testbed
// +build testbed

package testbed

import (
	"bytes"
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	crypto "github.com/dusk-network/dusk-crypto/hash"
	log "github.com/sirupsen/logrus"
)

var (
	clusterSize      = os.Getenv("CLUSTER_SIZE")
	dummyPayloadSize = os.Getenv("MSG_SIZE")
	ruskExecutable   = os.Getenv("RUSK_PATH")

	NetworkAddr    = "127.0.0.1"
	bootstrapNodes = []string{NetworkAddr + ":20000", NetworkAddr + ":20001"}

	baseKadcastPort = 20000
	baseGRPCPort    = 9000
)

func bootstrapCluster(ctx context.Context, t *testing.T) []*testNode {
	log.WithField("cluster_size", clusterSize).
		WithField("rusk", ruskExecutable).
		WithField("message_size", dummyPayloadSize).
		Info("Bootstrap cluster")

	size, err := strconv.Atoi(clusterSize)
	if err != nil {
		t.Fatal(err)
	}

	cluster := make([]*testNode, 0)

	for i := 0; i < size; i++ {
		n, err := newNode(ruskExecutable, bootstrapNodes, baseKadcastPort+i, baseGRPCPort+i)
		if err != nil {
			t.Error(err)
		}

		switch i {
		case 1:
			// Allow bootstrappers to start up
			time.Sleep(5 * time.Second)
		default:
			// Allow each node to complete start-up procedure
			time.Sleep(1 * time.Second)
		}

		// Start a listner for this node
		go n.Listen(ctx)

		cluster = append(cluster, n)
	}

	time.Sleep(5 * time.Second)
	return cluster
}

func assertBroadcastMsgReceived(t *testing.T, cluster []*testNode, sender int, d time.Duration) {
	// Node 0 broadcast a message of dummyPayloadSize
	msgSize, err := strconv.Atoi(dummyPayloadSize)
	if err != nil {
		panic(err)
	}

	blob, _ := crypto.RandEntropy(uint32(msgSize))
	cluster[sender].Broadcast(context.Background(), blob)

	// Ensure the entire network received the message, except the initiator
	time.Sleep(d)

	for i, tn := range cluster {
		if i == sender {
			continue
		}

		msg := tn.GetMessage()
		if !bytes.Equal(msg, blob) {
			t.Errorf("not equal at node %d, recv_msg_len: %d", tn.kadcastPort, len(msg))
		}
	}
}

func TestCluster(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Setup network
	cluster := bootstrapCluster(ctx, t)

	// Broadcast a message from node_0
	assertBroadcastMsgReceived(t, cluster, 0, 10*time.Second)

	// teardown
	cancel()
	log.Info("canceling ...")
	time.Sleep(1 * time.Second)

	for _, tn := range cluster {
		tn.Kill()
	}
}