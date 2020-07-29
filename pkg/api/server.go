package api

import (
	"net"
	"net/http"
	"time"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/capi"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"

	"github.com/etherlabsio/healthcheck"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/gorilla/pat"

	"github.com/sirupsen/logrus"

	"context"
)

var (
	router *pat.Router
	log    = logrus.WithField("package", "api")
)

// Server defines the HTTP server of the API
type Server struct {
	started bool // Indicates whether or not server has started

	listener net.Listener

	// Node components
	eventBus *eventbus.EventBus
	rpcBus   *rpcbus.RPCBus
	db       database.DB

	Server *http.Server
}

//NewHTTPServer return pointer to new created server object
func NewHTTPServer(eventBus *eventbus.EventBus, rpcBus *rpcbus.RPCBus, memoryDB database.DB) (*Server, error) {
	srv := Server{
		eventBus: eventBus,
		rpcBus:   rpcBus,
		db:       memoryDB,
	}
	router = srv.InitRouting()
	httpServer := &http.Server{
		Addr:    cfg.Get().API.Address,
		Handler: router,
	}
	srv.Server = httpServer

	return &srv, nil
}

//Start will start and and listen the *http.Server
func (s *Server) Start(srv *Server) error {

	log.WithField("address", cfg.Get().API.Address).Info("Starting API server")

	//enable graceful shutdown
	err := gracehttp.Serve(
		srv.Server,
	)

	return err
}

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
	capi.StartAPI(s.eventBus, s.rpcBus, s.db)

	r.HandleFunc("/consensus/bidders", capi.GetBidders).Methods("GET")
	r.HandleFunc("/consensus/provisioners", capi.GetProvisioners).Methods("GET")
	r.HandleFunc("/consensus/currentstep", capi.GetCurrentStep).Methods("GET")
	r.HandleFunc("/consensus/eventqueuestatus", capi.GetEventQueueStatus).Methods("GET")

	return r
}
