package rpc

import (
	"bytes"
	"encoding/json"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"strconv"
	"time"
)

// handler defines a method bound to an RPC command.
type handler func(*Server, []string) (string, error)

// rpcCmd maps method names to their actual functions.
var rpcCmd = map[string]handler{
	"version":       version,
	"ping":          pong,
	"uptime":        uptime,
	"chaininfo":     chainInfo,
	"getlastblock":  getlastblock,
	"getmempooltxs": getmempooltxs,
}

// rpcAdminCmd holds all admin methods.
var rpcAdminCmd = map[string]bool{}

// version will return the version of the client.
var version = func(s *Server, params []string) (string, error) {
	return protocol.NodeVer.String(), nil
}

// pong simply returns "pong" to let the caller know the server is up.
var pong = func(s *Server, params []string) (string, error) {
	return "pong", nil
}

// uptime returns the server uptime.
var uptime = func(s *Server, params []string) (string, error) {
	return strconv.FormatInt(time.Now().Unix()-s.startTime, 10), nil
}

var chainInfo = func(s *Server, params []string) (string, error) {
	// ask blockchain for info
	s.eventBus.Publish(string(topics.RPCChainInfo), nil)

	// wait for blockchain to reply
	m := <-s.decodedChainInfoChannel

	return m, nil
}

var getlastblock = func(s *Server, params []string) (string, error) {

	r, err := s.rpcBus.Call(wire.GetLastBlock, wire.NewRequest(bytes.Buffer{}, 1))

	if err != nil {
		return "", err
	}

	b := &block.Block{}
	err = b.Decode(&r)

	if err != nil {
		return "", err
	}

	res, err := json.Marshal(b)

	return string(res), err
}

var getmempooltxs = func(s *Server, params []string) (string, error) {

	r, err := s.rpcBus.Call(wire.GetVerifiedTxs, wire.NewRequest(bytes.Buffer{}, 1))

	if err != nil {
		return "", err
	}

	lTxs, err := encoding.ReadVarInt(&r)
	if err != nil {
		return "", err
	}

	txs, err := transactions.FromReader(&r, lTxs)
	if err != nil {
		return "", err
	}

	res, err := json.Marshal(txs)

	return string(res), err
}
