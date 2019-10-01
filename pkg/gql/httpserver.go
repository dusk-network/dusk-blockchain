package gql

import (
	"crypto/tls"
	"encoding/base64"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/gql/query"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/graphql-go/graphql"

	logger "github.com/sirupsen/logrus"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	cryptoutils "github.com/dusk-network/dusk-blockchain/pkg/util/crypto"
	"golang.org/x/crypto/sha3"
)

var log *logger.Entry = logger.WithFields(logger.Fields{"process": "gql"})

// Server defines the HTTP server of the GraphQL service node.
type Server struct {
	started   bool // Indicates whether or not server has started
	eventBus  *eventbus.EventBus
	rpcBus    *rpcbus.RPCBus
	db        database.DB
	authSHA   []byte
	listener  net.Listener
	startTime int64

	schema *graphql.Schema
}

// NewHTTPServer instantiates a new NewHTTPServer to handle GraphQL queries.
func NewHTTPServer(eventBus *eventbus.EventBus, rpcBus *rpcbus.RPCBus) (*Server, error) {

	srv := Server{
		eventBus: eventBus,
		rpcBus:   rpcBus,
	}

	user := cfg.Get().Gql.User
	pass := cfg.Get().Gql.Pass
	if user != "" && pass != "" {
		login := user + ":" + pass
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
		authSHA := sha3.Sum256([]byte(auth))

		srv.authSHA = authSHA[:]
	}

	return &srv, nil
}

// Start the GraphQL HTTP Server and begin listening on specified port.
func (s *Server) Start() error {
	ServeMux := http.NewServeMux()
	httpServer := &http.Server{
		Handler:     ServeMux,
		ReadTimeout: time.Second * 10,
	}

	network := cfg.Get().Gql.Network
	address := cfg.Get().Gql.Address

	// GraphQL serving over HTTP
	ServeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Connection", "close")
		w.Header().Set("Content-Type", "application/json")
		r.Close = true

		if s.started {
			handleQuery(s.schema, w, *r, s.db)
		} else {
			log.Warn("HTTP service is not running")
		}
	})

	//  Setup graphQL
	rootQuery := query.NewRoot(s.rpcBus)
	sc, err := graphql.NewSchema(
		graphql.SchemaConfig{Query: rootQuery.Query},
	)

	if err != nil {
		return err
	}

	s.schema = &sc
	_, s.db = heavy.CreateDBConnection()

	if network == "unix" {
		if err := os.RemoveAll(address); err != nil {
			return err
		}
	}

	// Define TLS configuration
	certFile := cfg.Get().Gql.CertFile
	keyFile := cfg.Get().Gql.KeyFile
	tlsConfig, err := cryptoutils.GenerateTLSServerConfig(certFile, keyFile)
	if err != nil {
		return err
	}

	// Set up tls listener socket
	var bindSocket string
	if network == "unix" {
		bindSocket = cfg.Get().Gql.Address
	} else {
		bindSocket = cfg.Get().Gql.Address + ":" + cfg.Get().Gql.Port
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
	s.startTime = time.Now().Unix()

	return nil
}

// Listen on the http server.
func (s *Server) listenOnHTTPServer(httpServer *http.Server) {

	log.Infof("HTTP server listening on port %v", cfg.Get().Gql.Port)

	if err := httpServer.Serve(s.listener); err != http.ErrServerClosed {
		log.Errorf("HTTP server stopped with error %v", err)
	} else {
		log.Info("HTTP server stopped listening")
	}
}

// Stop the server
func (s *Server) Stop() error {
	s.started = false
	if err := s.listener.Close(); err != nil {
		log.Errorf("error shutting down, %v\n", err)
		return err
	}

	return nil
}
