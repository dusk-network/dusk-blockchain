package rpc

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

// JSONRequest defines a JSON-RPC request.
type JSONRequest struct {
	Method string   `json:"method"`
	Params []string `json:"params,omitempty"`
}

// JSONResponse defines a JSON-RPC response to a method call.
type JSONResponse struct {
	Result string `json:"result"`
	Error  string `json:"error"`
}

// handleRequest takes a JSON-RPC request and parses it, then returns the result to the
// message sender.
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request, isAdmin bool) {
	// Only handle requests if server is started
	if !s.started {
		return
	}

	// Read and close JSON request body
	body, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		http.Error(w, fmt.Sprintf("%d error reading JSON: %v", http.StatusBadRequest, err),
			http.StatusBadRequest)
		return
	}

	var req JSONRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return
	}

	// Parse and run passed method
	result, err := s.runCmd(&req, isAdmin)
	if err != nil {
		// Request was unauthorized, so return a http.Error
		w.Header().Add("WWW-Authenticate", `Basic realm="duskd admin RPC"`)
		http.Error(w, err.Error(), http.StatusUnauthorized)
	}

	if _, err := w.Write([]byte(result)); err != nil {
		fmt.Fprintf(os.Stderr, "error writing reply: %v\n", err)
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
