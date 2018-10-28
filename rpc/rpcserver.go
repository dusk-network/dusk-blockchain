package rpc

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/toghrulmaharramov/dusk-go/crypto/hash"
)

// Server provides a RPC server to the Dusk daemon
type Server struct {
	Started  bool         // Indicates whether or not server has started
	AuthSHA  []byte       // Hash of the auth credentials
	Config   Config       // Configuration struct for RPC server
	Listener net.Listener // RPC Server listener
	StopChan chan string  // Channel to quit the daemon on 'stopnode' command
}

// NewRPCServer instantiates a new RPCServer.
func NewRPCServer(cfg *Config) (*Server, error) {
	srv := Server{
		Config:   *cfg,
		StopChan: make(chan string),
	}

	if cfg.RPCUser != "" && cfg.RPCPassword != "" {
		login := cfg.RPCUser + ":" + cfg.RPCPassword
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
		authSHA, err := hash.Sha3256([]byte(auth))
		if err != nil {
			return nil, err
		}

		srv.AuthSHA = authSHA
	}

	return &srv, nil
}

// CheckAuth checks whether supplied credentials match the server credentials.
func (s *Server) CheckAuth(r *http.Request) (bool, error) {
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
func (s *Server) Start() error {
	s.Started = true
	ServeMux := http.NewServeMux()
	httpServer := &http.Server{
		Handler:     ServeMux,
		ReadTimeout: time.Second * 10,
	}

	// HTTP handler
	ServeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")
		w.Header().Set("Content-Type", "application/json")
		r.Close = true

		// Check authentication
		isAdmin, err := s.CheckAuth(r)
		if err != nil {
			// AuthFail(w)
		}

		s.HandleRequest(w, r, isAdmin)
	})

	// Set up listener
	l, err := net.Listen("tcp", "localhost:9999")
	if err != nil {
		return err
	}

	// Assign to Server
	s.Listener = l

	// Start listening
	go func(l net.Listener) {
		fmt.Fprintf(os.Stdout, "RPC server listening on port %v\n", s.Config.RPCPort)
		httpServer.Serve(l)
		fmt.Fprintf(os.Stdout, "RPC server stopped listening\n")
	}(s.Listener)

	return nil
}

// Stop will close the RPC server
func (s *Server) Stop() error {
	s.Started = false
	if err := s.Listener.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "error shutting down RPC, %v\n", err)
		return err
	}

	return nil
}
