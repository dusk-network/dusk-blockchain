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

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	pb "github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

// PublishTopic publishes an event bus topic to the specified node via
// rpc call
func (n *Network) PublishTopic(nodeIndex uint, topic, payload string) error {

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
	return nil
}

// SendQuery sends a graphql query to the specified network node
func (n *Network) SendQuery(nodeIndex uint, query string, result interface{}) error {
	if nodeIndex >= uint(len(n.Nodes)) {
		return errors.New("invalid node index")
	}

	targetNode := n.Nodes[nodeIndex]
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

// LoadWalletCmd sends gRPC command LoadWallet and returns pubkey (if loaded)
func (n *Network) LoadWalletCmd(ind uint, password string) (string, error) {

	addr := "unix://" + n.Nodes[ind].Cfg.RPC.Address

	// Set up a connection to the server.
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return "", err
	}
	defer func() {
		_ = conn.Close()
	}()

	client := pb.NewWalletClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := pb.LoadRequest{Password: password}
	resp, err := client.LoadWallet(ctx, &req)
	if err != nil {
		return "", err
	}

	return string(resp.GetKey().PublicKey), nil
}

// SendBidCmd sends gRPC command SendBid and returns tx hash
func (n *Network) SendBidCmd(ind uint, amount, locktime uint64) ([]byte, error) {

	addr := "unix://" + n.Nodes[ind].Cfg.RPC.Address

	// Set up a connection to the server.
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = conn.Close()
	}()

	client := pb.NewTransactorClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := pb.BidRequest{Amount: amount}
	resp, err := client.Bid(ctx, &req)
	if err != nil {
		return nil, err
	}

	return resp.Hash, nil
}

// SendWireMsg sends a P2P message to the specified network node
// NB: Handshaking procedure must be performed prior to the message sending
func (n *Network) SendWireMsg(ind uint, msg []byte, writeTimeout int) error {

	if ind >= uint(len(n.Nodes)) {
		return errors.New("invalid node index")
	}

	targetNode := n.Nodes[ind]
	addr := "127.0.0.1:" + targetNode.Cfg.Network.Port

	// connect to this socket
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	writeTimeoutDuration := time.Duration(writeTimeout) * time.Second
	_ = conn.SetWriteDeadline(time.Now().Add(writeTimeoutDuration))
	_, err = conn.Write(msg)

	return err
}

// ConstructWireFrame creates a frame according to the wire protocol
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

// WriteFrame writes a frame to a buffer
// TODO: remove *bytes.Buffer from the returned parameters
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

// WaitUntil blocks until the node at index ind reaches the target height
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
// Returns hash of the block that includes this tx
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

	// asserts that given condition will be met in 1 minute, by checking condition function each second.
	assert.Eventuallyf(t, condition, 1*time.Minute, time.Second, "failed node %s", ind)

	return blockhash
}
