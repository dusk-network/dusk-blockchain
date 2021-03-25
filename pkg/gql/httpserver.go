// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package gql

import (
	"context"
	"net"
	"net/http"
	"os"
	"sync/atomic"
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
	endpointWSS = "/wss"
	endpointGQL = "/graphql"
)

// Server defines the HTTP server of the GraphQL service node.
type Server struct {
	started uint32 // Indicates whether or not server has started

	httpServer *http.Server
	lmt        *limiter.Limiter

	// Graphql utility.
	schema *graphql.Schema

	// Websocket connections pool.
	pool *notifications.BrokerPool

	// Node components.
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
	s.httpServer = &http.Server{
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

	go s.listenOnHTTPServer(l)

	atomic.StoreUint32(&s.started, 1)
	return nil
}

// Listen on the http server.
func (s *Server) listenOnHTTPServer(l net.Listener) {
	conf := cfg.Get().Gql

	log.WithField("net", conf.Network).
		WithField("addr", conf.Address).
		WithField("tls", conf.EnableTLS).Infof("GraphQL HTTP server listening")

	var err error
	if conf.EnableTLS {
		if _, err = os.Stat(conf.CertFile); err != nil {
			log.WithError(err).Fatal("failed to enable TLS")
		}

		err = s.httpServer.ServeTLS(l, conf.CertFile, conf.KeyFile)
	} else {
		err = s.httpServer.Serve(l)
	}

	if err != nil && err != http.ErrServerClosed {
		log.WithError(err).Warn("HTTP server stopped with error")
	} else {
		log.Info("HTTP server stopped listening")
	}
}

// EnableGraphQL sets up the GraphQL service, wires the request handler, sets
// the limiter, instantiates the Schema and creates a DB connection.
func (s *Server) EnableGraphQL(serverMux *http.ServeMux) error {
	// GraphQL service
	gqlHandler := func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadUint32(&s.started) == 0 {
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
// broker) to push graphql notifications over websocket.
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

	log.WithField("brokers_num", nc.BrokersNum).
		WithField("clients_per_broker", clientsPerBroker).
		Info("Start graphql notification service")

	s.pool = notifications.NewPool(s.eventBus, nc.BrokersNum, clientsPerBroker)

	wsHandler := func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadUint32(&s.started) == 0 {
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.WithError(err).Error("Failed to set websocket upgrade")
			return
		}

		s.pool.PushConn(conn)
	}

	middleware := tollbooth.LimitFuncHandler(s.lmt, wsHandler)

	endpoint := endpointWS
	if cfg.Get().Gql.EnableTLS {
		endpoint = endpointWSS
	}

	serverMux.Handle(endpoint, middleware)

	return nil
}

// Stop the server.
func (s *Server) Stop() error {
	atomic.StoreUint32(&s.started, 0)

	// Close pool of notification brokers
	s.pool.Close()

	// Close graphql http server
	dialCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(dialCtx); err != nil {
		log.WithError(err).Warn("error shutting down")
		return err
	}

	return nil
}
