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
}

// SendQuery sends a graphql query to the specified network node
func (n *Network) SendQuery(nodeIndex uint, query string, result interface{}) error {
	if nodeIndex >= uint(len(n.Nodes)) {
		return errors.New("invalid node index")
	}

	targetNode := n.Nodes[nodeIndex]
	addr := "http://" + targetNode.Cfg.Gql.Address

	buf := bytes.Buffer{}
	if _, err := buf.Write([]byte(query)); err != nil {
		return errors.New("invalid query")
	}

	resp, err := http.Post(addr, "application/json", &buf)
	if err != nil {
		return err
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return err
	}

	return nil
}

// SendCommand sends a jsonrpc request to the specified network node.
// Returns a string with the response of the json-rpc server.
func (n *Network) SendCommand(nodeIndex uint, method string, params []string) (string, error) {

	if nodeIndex >= uint(len(n.Nodes)) {
		return "", errors.New("invalid node index")
	}

	targetNode := n.Nodes[nodeIndex]

	req := rpc.JSONRequest{
		Method: method,
		Params: params,
	}

	var data []byte
	data, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	buf := bytes.Buffer{}
	if _, err := buf.Write(data); err != nil {
		return "", err
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

	request, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return "", err
	}

	request.SetBasicAuth(targetNode.Cfg.RPC.User, targetNode.Cfg.RPC.Pass)

	resp, err := httpc.Do(request)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var jsonResp rpc.JSONResponse
	if err := json.Unmarshal(body, &jsonResp); err != nil {
		return "", err
	}

	return jsonResp.Result, nil
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
