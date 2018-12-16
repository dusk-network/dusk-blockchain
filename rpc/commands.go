package rpc

import (
	"strconv"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/crypto/base58"

	"gitlab.dusk.network/dusk-core/dusk-go/crypto/hash"
)

// Handler defines a method bound to an RPC command.
type Handler func(*Server, []string) (string, error)

// RPCCmd maps method names to their actual functions.
var RPCCmd = map[string]Handler{
	"version":  Version,
	"ping":     Pong,
	"uptime":   Uptime,
	"stopnode": StopNode,
	"hash":     Hash,
}

// RPCAdminCmd holds all admin methods.
var RPCAdminCmd = map[string]bool{
	"stopnode": true,
}

// Version will return the version of the client.
var Version = func(s *Server, params []string) (string, error) {
	// In the future, set this to actually get version number from running daemon.
	// For now though, just return a string for testing purposes.
	return "0.1", nil
}

// StopNode will stop the RPC server and tell the daemon to shut down through
// the server's StopChan.
var StopNode = func(s *Server, params []string) (string, error) {
	if err := s.Stop(); err != nil {
		return "", err
	}

	s.StopChan <- "stop"
	return "Daemon exiting...", nil
}

// Pong simply returns "pong" to let the caller know the server is up.
var Pong = func(s *Server, params []string) (string, error) {
	return "pong", nil
}

// Uptime returns the server uptime.
var Uptime = func(s *Server, params []string) (string, error) {
	return strconv.FormatInt(time.Now().Unix()-s.StartTime, 10), nil
}

// Hash will return the SHA3-256 hash of each word passed.
// NOTE: this is a test function to see if parameter passing will work.
// This function does not need to stay and can be deleted.
var Hash = func(s *Server, params []string) (string, error) {
	var hashes string
	for _, word := range params {
		hash, err := hash.Sha3256([]byte(word))
		if err != nil {
			return "", err
		}

		text := base58.Base58Encoding(hash)
		hashes += text + " "
	}

	return hashes, nil
}
