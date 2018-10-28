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
	// ID string
	Method string
	Params []string
}

// JSONResponse defines a JSON-RPC response to a method call.
type JSONResponse struct {
	// ID string
	Result interface{}
	Error  error
}

// HandleRequest takes a JSON-RPC request and parses it, then returns the result to the
// message sender.
func (s *Server) HandleRequest(w http.ResponseWriter, r *http.Request, isAdmin bool) {
	// Only handle requests if server is started
	if !s.Started {
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
	result := s.RunCmd(&req)

	msg, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling response: %v", err)
	}

	if _, err := w.Write(msg); err != nil {
		fmt.Fprintf(os.Stderr, "error writing reply: %v", err)
		return
	}
}

// RunCmd parses and runs the specified method. Server is included as the receiver in case
// the method needs to modify anything on the RPC server.
func (s *Server) RunCmd(r *JSONRequest) *JSONResponse {
	resp := JSONResponse{}
	// Get method
	fn := RPCCmd[r.Method]

	// Run method and return result
	result, err := fn(s, r.Params)
	resp.Result = result
	resp.Error = err
	return &resp
}
