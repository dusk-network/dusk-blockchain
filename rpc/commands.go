package rpc

import "time"

// Handler defines a method bound to an RPC command.
type Handler func(*Server, []string) (interface{}, error)

// RPCCmd maps method names to their actual functions.
var RPCCmd = map[string]Handler{
	"version":  Version,
	"ping":     Pong,
	"uptime":   Uptime,
	"stopnode": StopNode,
}

// RPCAdminCmd holds all admin methods.
var RPCAdminCmd = map[string]bool{
	"stopnode": true,
}

// Version will return the version of the client.
var Version = func(s *Server, params []string) (interface{}, error) {
	// In the future, set this to actually get version number from running daemon.
	// For now though, just return a string for testing purposes.
	return "0.1", nil
}

// StopNode will stop the RPC server and tell the daemon to shut down through
// the server's StopChan.
var StopNode = func(s *Server, params []string) (interface{}, error) {
	if err := s.Stop(); err != nil {
		return nil, err
	}

	s.StopChan <- "stop"
	return "Daemon exiting...", nil
}

// Pong simply returns "pong" to let the caller know the server is up.
var Pong = func(s *Server, params []string) (interface{}, error) {
	return "pong", nil
}

// Uptime returns the server uptime.
var Uptime = func(s *Server, params []string) (interface{}, error) {
	return time.Now().Unix() - s.StartTime, nil
}
