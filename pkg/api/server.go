// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package api

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/capi"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"

	"github.com/etherlabsio/healthcheck"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/gorilla/pat"

	"github.com/sirupsen/logrus"
)

var (
	router *pat.Router
	log    = logrus.WithField("package", "api")
)

// Server defines the HTTP server of the API.
type Server struct {
	// Node components.
	eventBus *eventbus.EventBus
	rpcBus   *rpcbus.RPCBus
	store    *capi.StormDBInstance

	Server *http.Server
}

// NewHTTPServer return pointer to new created server object.
func NewHTTPServer(eventBus *eventbus.EventBus, rpcBus *rpcbus.RPCBus) (*Server, error) {
	dbFile := cfg.Get().API.DBFile
	if dbFile == "" {
		log.Info("Will start monitoring db with in-memory since DBFile cfg is not set")

		dir, err := ioutil.TempDir(os.TempDir(), "storm")
		if err != nil {
			panic(err)
		}

		dbFile = filepath.Join(dir, "api.db")
	}

	store, err := capi.NewStormDBInstance(dbFile)
	if err != nil {
		log.Fatal(err)
	}

	capi.SetStormDBInstance(store)

	srv := Server{
		eventBus: eventBus,
		rpcBus:   rpcBus,
		store:    store,
	}

	addr := cfg.Get().API.Address
	log.WithField("addr", addr).Info("Will InitRouting for Consensus API server")

	router = srv.InitRouting()
	httpServer := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	srv.Server = httpServer
	return &srv, nil
}

// Start will start and and listen the *http.Server.
func (s *Server) Start(srv *Server) error {
	log.WithField("address", cfg.Get().API.Address).Info("Starting API server")

	// enable graceful shutdown
	err := gracehttp.Serve(
		srv.Server,
	)

	return err
}

// InitRouting will init pat.Router endpoints.
func (s *Server) InitRouting() *pat.Router {
	r := pat.New()

	r.Handle("/healthcheck", healthcheck.Handler(
		// WithTimeout allows you to set a max overall timeout.
		healthcheck.WithTimeout(5*time.Second),

		healthcheck.WithChecker(
			"status", healthcheck.CheckerFunc(
				func(ctx context.Context) error {
					return nil
				},
			),
		),
	))

	// init consensus API services
	capi.StartAPI(s.eventBus, s.rpcBus)

	r.HandleFunc("/consensus/provisioners", capi.GetProvisionersHandler).Methods("GET")
	r.HandleFunc("/consensus/roundinfo", capi.GetRoundInfoHandler).Methods("GET")
	r.HandleFunc("/consensus/eventqueuestatus", capi.GetEventQueueStatusHandler).Methods("GET")
	r.HandleFunc("/p2p/logs", capi.GetP2PLogsHandler).Methods("GET")
	r.HandleFunc("/p2p/count", capi.GetP2PCountHandler).Methods("GET")

	return r
}
