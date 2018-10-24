package rpc

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/toghrulmaharramov/dusk-go/crypto/hash"
	"net"
	"net/http"
	"os"
	"time"
)

// RPCServer provides a RPC server to the Dusk daemon
type RPCServer struct {
	Started  bool         // Indicates whether or not server has started
	AuthSHA  []byte       // Hash of the auth credentials
	Config   RPCConfig    // Configuration struct for RPC server
	Listener net.Listener // RPC Server listener
}

// NewRPCServer instantiates a new RPCServer.
func NewRPCServer(cfg *RPCConfig) (*RPCServer, error) {
	rpc := RPCServer{
		Config: *cfg,
	}

	if cfg.RPCUser != "" && cfg.RPCPassword != "" {
		login := cfg.RPCUser + ":" + cfg.RPCPassword
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
		authSHA, err := hash.Sha3256([]byte(auth))
		if err != nil {
			return nil, err
		}

		rpc.AuthSHA = authSHA
	}

	return &rpc, nil
}

// CheckAuth checks whether supplied credentials match the server credentials.
func (s *RPCServer) CheckAuth(r *http.Request) (bool, error) {
	authHeader := r.Header["Authorization"]
	if len(authHeader) <= 0 {
		return false, nil
	}

	authSHA, err := hash.Sha3256([]byte(authHeader[0]))
	if err != nil {
		return false, err
	}

	if cmp := subtle.ConstantTimeCompare(authSHA[:], s.AuthSHA[:]); cmp == 1 {
		return true, nil
	}

	return false, errors.New("RPC auth failure")
}

// AuthFail returns a message back to the caller indicated that authentication failed.
func AuthFail(w http.ResponseWriter) {
	w.Header().Add("WWW-Authenticate", `Basic realm="duskd RPC"`)
	http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
}

// Start the RPC Server and begin listening on specified port.
func (s *RPCServer) Start() error {
	s.Started = true
	RPCServeMux := http.NewServeMux()
	httpServer := &http.Server{
		Handler:     RPCServeMux,
		ReadTimeout: time.Second * 10,
	}

	// HTTP handler
	RPCServeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")
		w.Header().Set("Content-Type", "application/json")
		r.Close = true

		// Check authentication
		isAdmin, err := s.CheckAuth(r)
		if err != nil {
			AuthFail(w)
			return
		}

		s.HandleRequest(w, r, isAdmin)
	})

	// Set up listener
	l, err := net.Listen("tcp", ":9999")
	if err != nil {
		return err
	}

	s.Listener = l

	// Start listening
	go func(l net.Listener) {
		httpServer.Serve(l)
	}(s.Listener)

	return nil
}

func (s *RPCServer) Stop() error {
	s.Started = false
	if err := s.Listener.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "error shutting down RPC, %v", err)
		return err
	}

	return nil
}
