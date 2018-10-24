package rpc

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func (s *RPCServer) HandleRequest(w http.ResponseWriter, r *http.Request, isAdmin bool) {
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

	// TODO: Parse JSON body into RPC request
}