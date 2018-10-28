package main

import "encoding/json"

// Request holds a JSON-RPC request.
type Request struct {
	// ID string
	Method string
	Params []string
}

// Response defines a response from the RPC server.
type Response struct {
	// ID string
	Result interface{}
	Error  error
}

// MarshalCmd simply puts the passed parameters into a Request struct
// and passes it to a json.Marshal function call.
func MarshalCmd(method string, params []string) ([]byte, error) {
	req := Request{
		Method: method,
		Params: params,
	}

	return json.Marshal(req)
}
