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

var log *logger.Entry = logger.WithFields(logger.Fields{"process": "gql"})

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

	if err := s.EnableGraphQL(mux); err != nil {
		return err
	}

	nc := cfg.Get().Gql.Notification
	if nc.BrokersNum > 0 {
		if err := s.EnableNotifications(mux); err != nil {
			return err
		}
	}

	// Set up HTTP Server over TCP
	l, err := net.Listen("tcp", cfg.Get().Gql.Address)
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

	g := cfg.Get().Gql
	log.Infof("HTTP server listening on %v", g.Address)

	var err error
	if g.EnableTLS {
		err = httpServer.ServeTLS(s.listener, g.CertFile, g.KeyFile)
	} else {
		err = httpServer.Serve(s.listener)
	}

	if err != nil && err != http.ErrServerClosed {
		log.Errorf("HTTP server stopped with error %v", err)
	} else {
		log.Info("HTTP server stopped listening")
	}
}

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
		log.Errorf("error shutting down, %v\n", err)
		return err
	}

	return nil
}
