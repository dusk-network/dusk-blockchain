// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	pb "github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

//nolint
// PublishTopic publishes an event bus topic to the specified node via
// rpc call.
func (n *Network) PublishTopic(nodeIndex uint, topic, payload string) error {
	return nil
	/*
		if nodeIndex >= uint(len(n.Nodes)) {
			return errors.New("invalid node index")
		}

		targetNode := n.Nodes[nodeIndex]
		addr := "http://127.0.0.1:" + targetNode.Cfg.RPC.Address
		request := &rpc.JSONRequest{Method: "publishTopic"}
		request.Params = []string{topic, payload}

		data, err := json.Marshal(request)
		if err != nil {
			return err
		}

		buf := bytes.Buffer{}
		if _, err := buf.Write(data); err != nil {
			return err
		}

		_, err = http.Post(addr, "application/json", &buf)
		return err
	*/
}

// SendQuery sends a graphql query to the specified network node.
func (n *Network) SendQuery(nodeIndex uint, query string, result interface{}) error {
	if nodeIndex >= uint(len(n.nodes)) {
		return errors.New("invalid node index")
	}

	targetNode := n.nodes[nodeIndex]
	addr := "http://" + targetNode.Cfg.Gql.Address + "/graphql"

	buf := bytes.Buffer{}
	if _, err := buf.Write([]byte(query)); err != nil {
		return errors.New("invalid query")
	}

	//nolint:gosec
	resp, err := http.Post(addr, "application/json", &buf)
	if err != nil {
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return err
	}

	return nil
}

// SendBidCmd sends gRPC command SendBid and returns tx hash.
func (n *Network) SendBidCmd(ind uint, amount, locktime uint64) ([]byte, error) {
	// session has been setup already in the TestMain, so here the client is
	// returning the permanent connection
	c := n.grpcClients[n.nodes[ind].Id]

	conn, err := c.GetSessionConn(grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, err
	}

	client := pb.NewTransactorClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := pb.BidRequest{Amount: amount}

	resp, err := client.Bid(ctx, &req)
	if err != nil {
		return nil, err
	}

	return resp.Hash, nil
}

// SendWireMsg sends a P2P message to the specified network node. Message should
// be in form of topic id + marshaled payload.
//
// The utility sets up a valid inbound peer connection with a localnet node.
// After the handshake procedure, it writes the message to the Peer connection.
func (n *Network) SendWireMsg(ind uint, msg []byte, writeTimeout int) error {
	if ind >= uint(len(n.nodes)) {
		return errors.New("invalid node index")
	}

	targetNode := n.nodes[ind]
	addr := "127.0.0.1:" + targetNode.Cfg.Network.Port

	// connect to this socket
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		return err
	}

	gossip := protocol.NewGossip(protocol.TestNet)
	pConn := peer.NewConnection(conn, gossip)
	w := peer.NewWriter(pConn, nil)

	// Run handshake procedure
	if err = w.Connect(protocol.FullNode); err != nil {
		return err
	}

	// Build wire frame
	buf := bytes.NewBuffer(msg)
	if err = gossip.Process(buf); err != nil {
		return err
	}

	// Write to the Peer connection
	if _, err = w.Connection.Write(buf.Bytes()); err != nil {
		return err
	}

	return err
}

// ConstructWireFrame creates a frame according to the wire protocol.
func ConstructWireFrame(magic protocol.Magic, cmd topics.Topic, payload *bytes.Buffer) ([]byte, error) {
	// Write magic
	buf := magic.ToBuffer()
	// Write topic
	if err := topics.Write(&buf, cmd); err != nil {
		return nil, err
	}

	// Write payload
	if _, err := buf.ReadFrom(payload); err != nil {
		return nil, err
	}

	// Build frame
	frame, err := WriteFrame(&buf)
	if err != nil {
		return nil, err
	}

	return frame.Bytes(), nil
}

// WriteFrame writes a frame to a buffer.
// TODO: remove *bytes.Buffer from the returned parameters.
func WriteFrame(buf *bytes.Buffer) (*bytes.Buffer, error) {
	msg := new(bytes.Buffer)
	// Append prefix(header)
	if err := encoding.WriteUint64LE(msg, uint64(0)); err != nil {
		return nil, err
	}

	// Append payload
	_, err := msg.ReadFrom(buf)
	if err != nil {
		return nil, err
	}

	// TODO: Append Checksum

	return msg, nil
}

// WaitUntil blocks until the node at index ind reaches the target height.
func (n *Network) WaitUntil(t *testing.T, ind uint, targetHeight uint64, waitFor time.Duration, tick time.Duration) {
	condition := func() bool {
		// Construct query to fetch block height
		query := "{\"query\" : \"{ blocks (last: 1) { header { height } } }\"}"

		var result map[string]map[string][]map[string]map[string]int
		if err := n.SendQuery(ind, query, &result); err != nil {
			return false
		}

		if result["data"]["blocks"][0]["header"]["height"] >= int(targetHeight) {
			return true
		}

		return false
	}

	assert.Eventually(t, condition, waitFor, tick)
}

// WaitUntilTx blocks until the node at index ind accepts the specified Tx
// Returns hash of the block that includes this tx.
func (n *Network) WaitUntilTx(t *testing.T, ind uint, txID string) string {
	var blockhash string

	condition := func() bool {
		// Construct query to fetch txid
		query := fmt.Sprintf(
			"{\"query\" : \"{ transactions (txid: \\\"%s\\\") { txid blockhash } }\"}",
			txID)

		var resp map[string]map[string][]map[string]string
		if err := n.SendQuery(ind, query, &resp); err != nil {
			return false
		}

		result, ok := resp["data"]
		if !ok || len(result["transactions"]) == 0 {
			// graphql request processed but still txid not found
			return false
		}

		if result["transactions"][0]["txid"] != txID {
			return false
		}

		blockhash = result["transactions"][0]["blockhash"]
		return true
	}

	// asserts that given condition will be met in 2 minutes, by checking
	// condition function each second.
	assert.Eventuallyf(t, condition, 2*time.Minute, time.Second, "failed node %s", ind)
	return blockhash
}

// SendStakeCmd sends gRPC command SendStake and returns tx hash.
func (n *Network) SendStakeCmd(ind uint, amount, locktime uint64) ([]byte, error) {
	// session has been setup already in the TestMain, so here the client is
	// returning the permanent connection
	c := n.grpcClients[n.nodes[ind].Id]

	conn, err := c.GetSessionConn(grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, err
	}

	client := pb.NewTransactorClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := pb.StakeRequest{Amount: amount, Locktime: locktime}

	resp, err := client.Stake(ctx, &req)
	if err != nil {
		return nil, err
	}

	return resp.Hash, nil
}

// GetLastBlockHeight makes an attempt to fetch last block height of a specified node.
func (n *Network) GetLastBlockHeight(ind uint) (uint64, error) {
	// Construct query to fetch block height
	query := "{\"query\" : \"{ blocks (last: 1) { header { height } } }\"}"

	var result map[string]map[string][]map[string]map[string]int
	if err := n.SendQuery(ind, query, &result); err != nil {
		return 0, err
	}

	return uint64(result["data"]["blocks"][0]["header"]["height"]), nil
}

// CalculateTPS makes an attempt to fetch last block height of a specified node.
func (n *Network) CalculateTPS(ind uint) (uint64, float32, int64, int, error) {
	// Fetch last height, timestamp and txs of the last two blocks
	query := "{\"query\" : \"{ blocks (last: 2) { header { height timestamp } transactions { txid } } }\"}"

	var result map[string]map[string][]map[string]interface{}
	if err := n.SendQuery(ind, query, &result); err != nil {
		return 0, 0, 0, 0, err
	}

	r := result["data"]

	if len(r["blocks"]) < 2 {
		return 0, 0, 0, 0, errors.New("not found")
	}

	last := r["blocks"][1]
	lastHeight := last["header"].(map[string]interface{})["height"].(float64)
	lastTimestamp := last["header"].(map[string]interface{})["timestamp"].(string)
	lastTxsCount := len(last["transactions"].([]interface{}))

	prev := r["blocks"][0]
	prevTimestamp := prev["header"].(map[string]interface{})["timestamp"].(string)

	lt, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", lastTimestamp)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	pt, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", prevTimestamp)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	blockTime := float32(lt.Unix() - pt.Unix())
	tps := float32(0)

	if blockTime > 0 {
		tps = float32(lastTxsCount) / blockTime
	}

	return uint64(lastHeight), tps, int64(blockTime), lastTxsCount, nil
}

// IsSynced checks if each node blockchain tip is close to the blockchain tip of node 0.
// threshold param is the number of blocks the last block can differ.
func (n *Network) IsSynced(threshold uint64) (uint64, error) {
	forks := make(map[uint64][]int)
	primaryHeight := uint64(0)

	for nodeID := 0; nodeID < n.Size(); nodeID++ {
		h, err := n.GetLastBlockHeight(uint(nodeID))
		if err != nil {
			logrus.WithField("nodeID", nodeID).WithError(err).Warn("fetch last block height failed")
			continue
		}

		if len(forks) == 0 {
			primaryHeight = h
			forks[h] = make([]int, 1)
			forks[h][0] = nodeID
			continue
		}

		// Check if this node tip is close to the network tip
		var nofork bool

		for tip := range forks {
			lowerBound := tip - threshold
			if lowerBound > tip {
				lowerBound = 0
			}

			upperBound := tip + threshold

			if h >= lowerBound && h < upperBound {
				// Good. Within the range
				forks[tip] = append(forks[tip], nodeID)
				nofork = true
				break
			}
		}

		if nofork {
			continue
		}

		// Bad. Not in the range
		// If the difference is higher than the specified threshold, we assume a forking has happened
		// NB Such situation could be observed also in case of a subset of the network getting stalled
		forks[h] = make([]int, 1)
		forks[h][0] = nodeID
	}

	// If more than 1 forks are found,
	if len(forks) > 1 {
		// Trace network status
		for tip := range forks {
			logMsg := fmt.Sprintf("Network at round [%d] driven by nodes [", tip)
			for _, nodeID := range forks[tip] {
				logMsg += fmt.Sprintf(" %d,", nodeID)
			}

			logMsg += "]"
			logrus.WithField("process", "monitor").Info(logMsg)
		}

		return 0, errors.New("network inconsistency detected")
	}

	if len(forks) == 0 {
		return 0, errors.New("network unreachable")
	}

	return primaryHeight, nil
}

// GetWalletAddress makes an attempt to get wallet address of a specified node.
func (n *Network) GetWalletAddress(ind uint) (string, []byte, error) {
	c := n.grpcClients[n.nodes[ind].Id]

	conn, err := c.GetSessionConn(grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return "", nil, err
	}

	client := pb.NewWalletClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetAddress(ctx, &node.EmptyRequest{})
	if err != nil {
		return "", nil, err
	}

	return string(resp.Key.PublicKey[0:10]) + "...", resp.Key.PublicKey, nil
}

// GetBalance makes an attempt to get wallet balance of a specified node.
// Returns both UnlockedBalance and LockedBalance.
func (n *Network) GetBalance(ind uint) (uint64, uint64, error) {
	c := n.grpcClients[n.nodes[ind].Id]

	conn, err := c.GetSessionConn(grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return 0, 0, err
	}

	client := pb.NewWalletClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetBalance(ctx, &node.EmptyRequest{})
	if err != nil {
		return 0, 0, err
	}

	return resp.UnlockedBalance, resp.LockedBalance, nil
}

// PrintWalletsInfo prints wallet address and balance of all network nodes.
func (n *Network) PrintWalletsInfo(t *testing.T) {
	for i := uint(0); i < uint(n.Size()); i++ {
		addr, _, err := n.GetWalletAddress(i)
		if err != nil {
			logrus.WithField("node", i).WithError(err).Error("Could not get wallet address")
		} else {
			logrus.WithField("node", i).WithField("address", addr).
				Infof("Pubkey")
		}

		ub, lb, err := n.GetBalance(i)
		if err != nil {
			logrus.WithField("node", i).WithError(err).Error("Could not get wallet balance")
		} else {
			logrus.WithField("node", i).WithField("locked", lb).WithField("unlocked", ub).
				Info("Balance")
		}
	}
}

// SendTransferTxCmd sends gRPC command SendTransfer and returns tx hash.
func (n *Network) SendTransferTxCmd(senderNodeInd, recvNodeInd uint, amount, fee uint64) ([]byte, error) {
	// Get wallet address of sender
	senderAddr, _, err := n.GetWalletAddress(senderNodeInd)
	if err != nil {
		return nil, err
	}

	// Get wallet address of receiver
	recvAddr, pubKey, err := n.GetWalletAddress(recvNodeInd)
	if err != nil {
		return nil, err
	}

	logrus.
		WithField("s_wallet", senderAddr).
		WithField("r_wallet", recvAddr).
		WithField("amount", amount).
		WithField("fee", fee).
		Debug("Sending transfer")

	// Send Transfer grpc command
	c := n.grpcClients[n.nodes[senderNodeInd].Id]

	conn, err := c.GetSessionConn(grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, err
	}

	client := pb.NewTransactorClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := pb.TransferRequest{Amount: amount, Address: pubKey, Fee: fee}

	resp, err := client.Transfer(ctx, &req)
	if err != nil {
		return nil, err
	}

	return resp.Hash, nil
}

// BatchSendTransferTx sends a transfer call from node of index senderNodeInd to senderNodeInd+1 node.
func (n *Network) BatchSendTransferTx(t *testing.T, senderNodeInd uint, batchSize uint, amount, fee uint64, timeout time.Duration) error {
	recvNodeInd := senderNodeInd + 1
	if recvNodeInd >= uint(n.Size()) {
		recvNodeInd = 0
	}

	// Get wallet address of sender
	senderAddr, _, err := n.GetWalletAddress(senderNodeInd)
	if err != nil {
		return err
	}

	// Get wallet address of receiver
	recvAddr, pubKey, err := n.GetWalletAddress(recvNodeInd)
	if err != nil {
		return err
	}

	logrus.
		WithField("s_wallet", senderAddr).
		WithField("r_wallet", recvAddr).
		WithField("amount", amount).
		WithField("fee", fee).
		Debug("Sending transfer")

	// Send Transfer grpc command
	c := n.grpcClients[n.nodes[senderNodeInd].Id]

	conn, err := c.GetSessionConn(grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return err
	}

	defer func() {
		err = conn.Close()
		if err != nil {
			logrus.WithError(err).WithField("sender_index", senderNodeInd).Warn("Closing grpc")
		}
	}()

	client := pb.NewTransactorClient(conn)

	for i := uint(0); i < batchSize; i++ {
		req := pb.TransferRequest{Amount: amount, Address: pubKey, Fee: fee}

		clientDeadline := time.Now().Add(timeout)
		ctx, cancel := context.WithDeadline(context.Background(), clientDeadline)

		// 100TPS throughput
		time.Sleep(5 * time.Millisecond)

		_, err := client.Transfer(ctx, &req)
		if err != nil {
			logrus.WithField("sender_index", senderNodeInd).Error(err)
			cancel()
			return nil
		}

		cancel()
	}

	return nil
}

// MonitorTPS tries to measure network TPS each N seconds.
func (n *Network) MonitorTPS(delay time.Duration) {
	// Start monitoring TPS value of the network
	nodeInd := uint(0)

	lastHeight := uint64(0)
	cumulativeTxsCount := 0

	blockTimeSum := int64(0)
	blocksCount := int64(0)

	for {
		time.Sleep(delay)

		h, tps, blockTime, lastTxsCount, err := n.CalculateTPS(nodeInd)
		if err != nil {
			logrus.WithError(err).
				WithField("node_ind", nodeInd).
				Warn("calculate tps failed")

			// Failover to another node if this fails
			nodeInd++
			if nodeInd >= uint(n.Size()) {
				nodeInd = 0
			}

			continue
		}

		if lastHeight == h {
			// do not print if height is the same
			continue
		}

		cumulativeTxsCount += lastTxsCount
		lastHeight = h

		blocksCount++

		blockTimeSum += blockTime

		logrus.WithField("height", h).
			WithField("cumulative_txs_count", cumulativeTxsCount).
			WithField("average_block_time", blockTimeSum/blocksCount).
			Infof("Measured TPS %.2f (%d/%d)", tps, lastTxsCount, blockTime)
	}
}
