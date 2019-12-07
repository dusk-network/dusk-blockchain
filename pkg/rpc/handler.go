package rpc

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// JSONRequest defines a JSON-RPC request.
type JSONRequest struct {
	Method string   `json:"method"`
	Params []string `json:"params,omitempty"`
}

// JSONResponse defines a JSON-RPC response to a method call.
type JSONResponse struct {
	Result *json.RawMessage `json:"result"`
	Error  string           `json:"error"`
}

// handleRequest takes a JSON-RPC request and parses it, then returns the result to the
// message sender.
func (s *Server) handleRequest(w http.ResponseWriter, r http.Request, isAdmin bool) {
	// Only handle requests if server is started
	if !s.started {
		log.Warn("json-rpc service is not running")
		return
	}

	// Read and close JSON request body
	body, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		msg := fmt.Sprintf("%d error reading JSON: %v", http.StatusBadRequest, err)
		log.Error(msg)

		http.Error(w, msg,
			http.StatusBadRequest)
		return
	}

	log.Tracef("Request: %s", string(body))

	var req JSONRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Errorf("json.unmarshal request: %v", err)
		return
	}

	// Parse and run passed method
	result, err := s.runCmd(&req, isAdmin)
	errorDesc := "null"
	if err != nil {
		log.Errorf("%v", err)
		errorDesc = err.Error()
		result = "{}"
	}

	rawMessage := json.RawMessage([]byte(result))
	resp := JSONResponse{Result: &rawMessage, Error: errorDesc}
	resultData, err := json.MarshalIndent(resp, "", "\t")
	if err != nil {
		log.Errorf("marshal response: %v", err)
	}

	if _, err := w.Write([]byte(resultData)); err != nil {
		log.Errorf("write response: %v", err)
		return
	}
}

// runCmd parses and runs the specified method. Server is included as the receiver in case
// the method needs to modify anything on the RPC server.
func (s *Server) runCmd(r *JSONRequest, isAdmin bool) (string, error) {

	// Get method
	fn, ok := rpcCmd[r.Method]
	if !ok {
		return "", fmt.Errorf("method %s unrecognized", r.Method)
	}

	// Check if it is an admin-only method first if caller is not admin
	if !isAdmin && rpcAdminCmd[r.Method] {
		return "", fmt.Errorf("unauthorized call to method %v", r.Method)
	}

	// Run method and return result
	result, err := fn(s, r.Params)
	if err != nil {
		return "", err
	}

	return result, nil
}
