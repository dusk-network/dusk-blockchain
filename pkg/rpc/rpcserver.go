package rpc

import (
	"bytes"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"golang.org/x/crypto/sha3"
)

// Server defines the RPC server of the Dusk node.
type Server struct {
	started bool // Indicates whether or not server has started

	eventBus         *wire.EventBus
	rpcBus           *wire.RPCBus
	chainInfoChannel <-chan *bytes.Buffer
	chainInfoID      uint32

	authSHA  []byte       // Hash of the auth credentials
	listener net.Listener // RPC Server listener

	decodedChainInfoChannel chan string

	startTime int64
}

// NewRPCServer instantiates a new RPCServer.
func NewRPCServer(eventBus *wire.EventBus, rpcBus *wire.RPCBus) (*Server, error) {
	chainInfoChannel := make(chan *bytes.Buffer, 10)

	srv := Server{
		eventBus:         eventBus,
		rpcBus:           rpcBus,
		chainInfoChannel: chainInfoChannel,
	}

	chainInfoID := srv.eventBus.Subscribe(string(topics.ChainInfo), chainInfoChannel)
	srv.chainInfoID = chainInfoID

	user := cfg.Get().RPC.User
	pass := cfg.Get().RPC.Pass
	if user != "" && pass != "" {
		login := user + ":" + pass
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
		authSHA := sha3.Sum256([]byte(auth))

		srv.authSHA = authSHA[:]
	}

	return &srv, nil
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

		isAdmin := s.checkAuth(r)
		s.handleRequest(w, r, isAdmin)
	})

	// Set up listener
	l, err := net.Listen("tcp", "localhost:"+cfg.Get().RPC.Port)
	if err != nil {
		return err
	}

	// Assign to Server
	s.listener = l

	go s.listenOnHTTPServer(httpServer)

	s.started = true
	s.startTime = time.Now().Unix()

	go s.listenOnEventBus()

	return nil
}

// Listen on the http server.
func (s *Server) listenOnHTTPServer(httpServer *http.Server) {
	fmt.Fprintf(os.Stdout, "RPC server listening on port %v\n", cfg.Get().RPC.Port)
	httpServer.Serve(s.listener)
	fmt.Fprintf(os.Stdout, "RPC server stopped listening\n")
}

// Listen on the event bus for relevant topics.
func (s *Server) listenOnEventBus() {
	for {
		messageBytes := <-s.chainInfoChannel

		// TODO: decode and marshal to JSON
		// implement once chaininfo is implemented into blockchain.

		s.decodedChainInfoChannel <- string(messageBytes.Bytes())
	}
}

// checkAuth checks whether supplied credentials match the server credentials.
func (s *Server) checkAuth(r *http.Request) bool {
	authHeader := r.Header["Authorization"]
	if len(authHeader) <= 0 {
		return false
	}

	authSHA := sha3.Sum256([]byte(authHeader[0]))
	if cmp := subtle.ConstantTimeCompare(authSHA[:], s.authSHA[:]); cmp == 1 {
		return true
	}

	return false
}

// Stop the RPC server
func (s *Server) Stop() error {
	s.started = false
	if err := s.listener.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "error shutting down RPC, %v\n", err)
		return err
	}

	return nil
}
