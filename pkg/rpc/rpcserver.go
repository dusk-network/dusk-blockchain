package rpc

import (
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"net"
	"net/http"
	"os"
	"time"

	logger "github.com/sirupsen/logrus"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	cryptoutils "github.com/dusk-network/dusk-blockchain/pkg/util/crypto"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"golang.org/x/crypto/sha3"
)

var log *logger.Entry = logger.WithFields(logger.Fields{"process": "rpc"})

// Server defines the RPC server of the Dusk node.
type Server struct {
	started bool // Indicates whether or not server has started

	eventBus *eventbus.EventBus
	rpcBus   *rpcbus.RPCBus

	authSHA  []byte       // Hash of the auth credentials
	listener net.Listener // RPC Server listener
}

// NewRPCServer instantiates a new RPCServer.
func NewRPCServer(eventBus *eventbus.EventBus, rpcBus *rpcbus.RPCBus) (*Server, error) {

	srv := Server{
		eventBus: eventBus,
		rpcBus:   rpcBus,
	}

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

	network := cfg.Get().RPC.Network
	address := cfg.Get().RPC.Address

	// HTTP handler
	ServeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")
		w.Header().Set("Content-Type", "application/json")

		log.Tracef("Handle request method: %s", r.Method)

		r.Close = true

		isAdmin := s.checkAuth(r)
		s.handleRequest(w, *r, isAdmin)
	})

	if network == "unix" {
		if err := os.RemoveAll(address); err != nil {
			return err
		}
	}

	// Define TLS configuration
	certFile := cfg.Get().RPC.CertFile
	keyFile := cfg.Get().RPC.KeyFile
	tlsConfig, err := cryptoutils.GenerateTLSServerConfig(certFile, keyFile)
	if err != nil {
		return err
	}

	// Set up tls listener socket
	var bindSocket string
	if network == "unix" {
		bindSocket = address
	} else {
		bindSocket = address + ":" + cfg.Get().RPC.Port
	}

	// Start to listen on socket
	l, err := tls.Listen(network, bindSocket, tlsConfig)
	if err != nil {
		return err
	}

	// Assign to Server
	s.listener = l

	go s.listenOnHTTPServer(httpServer)

	s.started = true

	return nil
}

// Listen on the http server.
func (s *Server) listenOnHTTPServer(httpServer *http.Server) {
	log.Infof("RPC server listening on (%s) %s", cfg.Get().RPC.Network, cfg.Get().RPC.Address)

	if err := httpServer.Serve(s.listener); err != http.ErrServerClosed {
		log.Errorf("RPC server stopped with error %v", err)
	} else {
		log.Info("RPC server stopped listening")
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
		log.Errorf("error shutting down RPC, %v\n", err)
		return err
	}

	return nil
}
