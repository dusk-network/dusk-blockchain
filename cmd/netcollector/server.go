// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
)

// NoteArgs message payload.
type NoteArgs struct {
	// Unique identifier of the P2P wire message.
	MsgID string

	// Sender node IP:Port address.
	Addr string

	// Unix nano timestamp of the point in time the node reads the wire message.
	Recv_at uint64
}

// Response is JSON-RPC NoteService response.
type Response struct {
	Result string
}

// NoteService is JSON-RPC NoteService request.
type NoteService struct {
	d *database
}

// Say NoteService Say method handler.
func (t *NoteService) Say( /*req*/ _ *http.Request, args *NoteArgs /*result*/, _ *Response) error {
	return t.d.HandleNote(args)
}

func runJSONRPCServer(addr string, d *database) {
	rpcServer := rpc.NewServer()
	rpcServer.RegisterCodec(json.NewCodec(), "application/json")
	rpcServer.RegisterCodec(json.NewCodec(), "application/json;charset=UTF-8")

	s := &NoteService{d: d}
	if err := rpcServer.RegisterService(s, ""); err != nil {
		panic(err)
	}

	router := mux.NewRouter()
	router.Handle("/rpc", rpcServer)

	if err := http.ListenAndServe(addr, router); err != nil {
		panic(err)
	}
}
