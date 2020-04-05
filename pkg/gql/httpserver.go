package gql

import (
	"net"
	"net/http"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/gql/notifications"
	"github.com/dusk-network/dusk-blockchain/pkg/gql/query"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"

	logger "github.com/sirupsen/logrus"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
)

var log = logger.WithFields(logger.Fields{"prefix": "gql"})

const (
	endpointWS  = "/ws"
	endpointGQL = "/graphql"
)

// Server defines the HTTP server of the GraphQL service node.
type Server struct {
	started bool // Indicates whether or not server has started

	listener net.Listener
	lmt      *limiter.Limiter

	// Graphql utility
	schema *graphql.Schema

	// Websocket connections pool
	pool *notifications.BrokerPool

	// Node components
	eventBus *eventbus.EventBus
	rpcBus   *rpcbus.RPCBus
	db       database.DB
}

// NewHTTPServer instantiates a new NewHTTPServer to handle GraphQL queries.
func NewHTTPServer(eventBus *eventbus.EventBus, rpcBus *rpcbus.RPCBus) (*Server, error) {

	max := float64(cfg.Get().Gql.MaxRequestLimit)

	srv := Server{
		eventBus: eventBus,
		rpcBus:   rpcBus,
		lmt:      tollbooth.NewLimiter(max, nil),
	}

	return &srv, nil
}

// Start the GraphQL HTTP Server and begin listening on specified port.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	httpServer := &http.Server{
		Handler:     mux,
		ReadTimeout: time.Second * 10,
	}

	conf := cfg.Get().Gql
	if err := s.EnableGraphQL(mux); err != nil {
		return err
	}

	if conf.Notification.BrokersNum > 0 {
		if err := s.EnableNotifications(mux); err != nil {
			return err
		}
	}

	// Set up HTTP Server over TCP
	l, err := net.Listen(conf.Network, conf.Address)
	if err != nil {
		return err
	}

	s.listener = l
	go s.listenOnHTTPServer(httpServer)

	s.started = true

	return nil
}

// Listen on the http server.
func (s *Server) listenOnHTTPServer(httpServer *http.Server) {

	conf := cfg.Get().Gql

	log.WithField("net", conf.Network).
		WithField("addr", conf.Address).
		WithField("tls", conf.EnableTLS).Infof("GraphQL HTTP server listening")

	var err error
	if conf.EnableTLS {
		err = httpServer.ServeTLS(s.listener, conf.CertFile, conf.KeyFile)
	} else {
		err = httpServer.Serve(s.listener)
	}

	if err != nil && err != http.ErrServerClosed {
		log.Errorf("HTTP server stopped with error %v", err)
	} else {
		log.Info("HTTP server stopped listening")
	}
}

// EnableGraphQL sets up the GraphQL service, wires the request handler, sets
// the limiter, instantiates the Schema and creates a DB connection
func (s *Server) EnableGraphQL(serverMux *http.ServeMux) error {

	// GraphQL service
	gqlHandler := func(w http.ResponseWriter, r *http.Request) {

		if !s.started {
			return
		}

		w.Header().Set("Connection", "close")
		w.Header().Set("Content-Type", "application/json")
		r.Close = true

		handleQuery(s.schema, w, r, s.db)
	}

	middleware := tollbooth.LimitFuncHandler(s.lmt, gqlHandler)
	serverMux.Handle(endpointGQL, middleware)

	//  Setup graphQL
	rootQuery := query.NewRoot(s.rpcBus)
	sconf := graphql.SchemaConfig{Query: rootQuery.Query}

	sc, err := graphql.NewSchema(sconf)
	if err != nil {
		return err
	}

	s.schema = &sc
	_, s.db = heavy.CreateDBConnection()

	return nil
}

// EnableNotifications uses the configured amount of brokers and clients (per
// broker) to push graphql notifications over websocket
func (s *Server) EnableNotifications(serverMux *http.ServeMux) error {

	nc := cfg.Get().Gql.Notification

	upgrader := &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	var clientsPerBroker uint = 100
	if nc.ClientsPerBroker > 0 {
		clientsPerBroker = nc.ClientsPerBroker
	}

	s.pool = notifications.NewPool(s.eventBus, nc.BrokersNum, clientsPerBroker)

	wsHandler := func(w http.ResponseWriter, r *http.Request) {

		if !s.started {
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Errorf("Failed to set websocket upgrade: %v", err)
			return
		}
		s.pool.PushConn(conn)
	}

	middleware := tollbooth.LimitFuncHandler(s.lmt, wsHandler)
	serverMux.Handle(endpointWS, middleware)

	return nil
}

// Stop the server
func (s *Server) Stop() error {

	s.started = false
	s.pool.Close()
	if err := s.listener.Close(); err != nil {
		log.Errorf("error shutting down, %v", err)
		return err
	}

	return nil
}
