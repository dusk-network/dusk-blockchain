package main

import (
	"encoding/json"

	"gitlab.dusk.network/dusk-core/dusk-go/rpc"
)

// MarshalCmd simply puts the passed parameters into a Request struct
// and passes it to a json.Marshal function call.
func MarshalCmd(method string, params []string) ([]byte, error) {
	req := rpc.JSONRequest{
		Method: method,
		Params: params,
	}

	return json.Marshal(req)
}
