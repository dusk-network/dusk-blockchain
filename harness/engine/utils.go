package engine

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/dusk-network/dusk-blockchain/pkg/rpc"
	"io/ioutil"
	"net/http"
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
