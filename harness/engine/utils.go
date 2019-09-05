package engine

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/rpc"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

// PublishTopic publishes an event bus topic to the specified node via
// rpc call
func (n *Network) PublishTopic(nodeIndex uint, topic, payload string) error {

	if nodeIndex >= uint(len(n.Nodes)) {
		return errors.New("invalid node index")
	}

	targetNode := n.Nodes[nodeIndex]
	addr := "http://127.0.0.1:" + targetNode.Cfg.RPC.Port
	request := &rpc.JSONRequest{Method: "publishTopic", Id: 1}
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
	json.NewDecoder(resp.Body).Decode(&decoded)

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
	addr := "http://127.0.0.1:" + targetNode.Cfg.RPC.Port

	req := rpc.JSONRequest{
		Method: method,
		Params: params,
		Id:     1,
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

	resp, err := http.Post(addr, "application/json", &buf)
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

// SendCommand sends a jsonrpc request to the specified network node
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
	if err := encoding.WriteUint32(buf, binary.LittleEndian, uint32(magic)); err != nil {
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
	if err := encoding.WriteUint64(msg, binary.LittleEndian, uint64(0)); err != nil {
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
