package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/rpc"
)

// SendPostRequest is a simple function to send POST request to RPC server and handle the response
func SendPostRequest(JSON []byte, cfg *rpc.Config) (*rpc.JSONResponse, error) {
	// Generate a request to the daemon RPC server
	url := "http://localhost:" + cfg.RPCPort
	body := bytes.NewReader(JSON)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Close = true
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(cfg.RPCUser, cfg.RPCPass)

	// Create client
	client := http.Client{}

	// Submit request
	httpResp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// Read the response bytes
	respBytes, err := ioutil.ReadAll(httpResp.Body)
	httpResp.Body.Close()
	if err != nil {
		return nil, err
	}

	// Handle error codes
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		if len(respBytes) == 0 {
			return &rpc.JSONResponse{
				Result: "error",
				Error:  fmt.Sprintf("%d %s", httpResp.StatusCode, http.StatusText(httpResp.StatusCode)),
			}, nil
		}
		return &rpc.JSONResponse{
			Result: "error",
			Error:  fmt.Sprintf("%d %v: %s", httpResp.StatusCode, http.StatusText(httpResp.StatusCode), respBytes),
		}, nil
	}

	// Unmarshal response
	var resp rpc.JSONResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
