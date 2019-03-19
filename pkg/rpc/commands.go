package rpc

import (
	"strconv"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// handler defines a method bound to an RPC command.
type handler func(*Server, []string) (string, error)

// rpcCmd maps method names to their actual functions.
var rpcCmd = map[string]handler{
	"version":   version,
	"ping":      pong,
	"uptime":    uptime,
	"chaininfo": chainInfo,
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
