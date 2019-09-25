package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/rpc"
)

// PublishTopic publishes an event bus topic to the specified node via
// rpc call
func (n *Network) PublishTopic(nodeIndex uint, topic, payload string) error {

	if nodeIndex >= uint(len(n.Nodes)) {
		return errors.New("invalid node index")
	}

	targetNode := n.Nodes[nodeIndex]
	addr := "http://127.0.0.1:" + targetNode.Cfg.RPC.Address
	request := &rpc.JSONRequest{Method: "publishTopic", ID: 1}
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
}

// SendQuery sends a graphql query to the specified network node
func (n *Network) SendQuery(nodeIndex uint, query string) (map[string][]map[string]string, error) {

	if nodeIndex >= uint(len(n.Nodes)) {
		return nil, errors.New("invalid node index")
	}

	targetNode := n.Nodes[nodeIndex]
	addr := "http://127.0.0.1:" + targetNode.Cfg.Gql.Port

	buf := bytes.Buffer{}
	if _, err := buf.Write([]byte(query)); err != nil {
		return nil, errors.New("invalid query")
	}

	resp, err := http.Post(addr, "application/json", &buf)
	if err != nil {
		return nil, err
	}

	var decoded map[string]map[string][]map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}

	result, ok := decoded["data"]
	if !ok {
		return nil, errors.New("missing data field")
	}

	return result, nil
}

// SendCommand sends a jsonrpc request to the specified network node
func (n *Network) SendCommand(nodeIndex uint, method string, params []string) ([]byte, error) {

	if nodeIndex >= uint(len(n.Nodes)) {
		return nil, errors.New("invalid node index")
	}

	targetNode := n.Nodes[nodeIndex]

	req := rpc.JSONRequest{
		Method: method,
		Params: params,
		ID:     1,
	}

	var data []byte
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	buf := bytes.Buffer{}
	if _, err := buf.Write([]byte(data)); err != nil {
		return nil, err
	}

	addr := targetNode.Cfg.RPC.Address
	network := targetNode.Cfg.RPC.Network

	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
	}

	url := "http://" + addr
	if network == "unix" {
		url = "http://unix" + addr
	}

	resp, err := httpc.Post(url, "application/json", &buf)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var jsonResp rpc.JSONResponse
	if err := json.Unmarshal(body, &jsonResp); err != nil {
		return nil, err
	}

	// TODO: Simplify
	data, err = jsonResp.Result.MarshalJSON()
	if err != nil {
		return nil, err
	}

	return data, nil
}

// SendWireMsg sends a P2P message to the specified network node
// NB: Handshaking procedure must be performed prior to the message sending
func (n *Network) SendWireMsg(nodeIndex uint, msg []byte, writeTimeout int) error {

	if nodeIndex >= uint(len(n.Nodes)) {
		return errors.New("invalid node index")
	}

	targetNode := n.Nodes[nodeIndex]
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

func ConstructWireFrame(magic protocol.Magic, cmd topics.Topic, payload *bytes.Buffer) ([]byte, error) {
	// Write magic
	buf := new(bytes.Buffer)
	if err := encoding.WriteUint32LE(buf, uint32(magic)); err != nil {
		return nil, err
	}

	// Write topic
	topicBytes := topics.TopicToByteArray(cmd)
	if _, err := buf.Write(topicBytes[:]); err != nil {
		return nil, err
	}

	// Write payload
	if _, err := buf.ReadFrom(payload); err != nil {
		return nil, err
	}

	// Build frame
	frame, err := WriteFrame(buf)
	if err != nil {
		return nil, err
	}

	return frame.Bytes(), nil
}

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
