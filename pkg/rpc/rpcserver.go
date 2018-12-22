package rpc

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/sha3"
)

// Server provides a RPC server to the Dusk daemon
type Server struct {
	Started   bool         // Indicates whether or not server has started
	AuthSHA   []byte       // Hash of the auth credentials
	Config    Config       // Configuration struct for RPC server
	Listener  net.Listener // RPC Server listener
	StartTime int64        // Time when the RPC server started up
	StopChan  chan string  // Channel to quit the daemon on 'stopnode' command
}

// NewRPCServer instantiates a new RPCServer.
func NewRPCServer(cfg *Config) (*Server, error) {
	srv := Server{
		Config:   *cfg,
		StopChan: make(chan string),
	}

	if cfg.RPCUser != "" && cfg.RPCPass != "" {
		login := cfg.RPCUser + ":" + cfg.RPCPass
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
		authSHA := sha3.Sum256([]byte(auth))

		srv.AuthSHA = authSHA[:]
	}

	return &srv, nil
}

// CheckAuth checks whether supplied credentials match the server credentials.
func (s *Server) CheckAuth(r *http.Request) bool {
	authHeader := r.Header["Authorization"]
	if len(authHeader) <= 0 {
		return false
	}

	authSHA := sha3.Sum256([]byte(authHeader[0]))
	if cmp := subtle.ConstantTimeCompare(authSHA[:], s.AuthSHA[:]); cmp == 1 {
		return true
	}

	return false
}

// Start the RPC Server and begin listening on specified port.
func (s *Server) Start() error {
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
		isAdmin := s.CheckAuth(r)
		s.HandleRequest(w, r, isAdmin)
	})

	// Set up listener
	l, err := net.Listen("tcp", "localhost:"+s.Config.RPCPort)
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

	s.Started = true

	// Mark start time
	s.StartTime = time.Now().Unix()

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
