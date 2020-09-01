package api

import (
	"net"
	"net/http"
	"time"

	"github.com/tidwall/buntdb"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/capi"
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

	Server *http.Server
}

//NewHTTPServer return pointer to new created server object
func NewHTTPServer(eventBus *eventbus.EventBus, rpcBus *rpcbus.RPCBus) (*Server, error) {

	dbFile := cfg.Get().API.DBFile
	if dbFile == "" {
		log.Info("Will start monitoring db with in-memory since DBFile cfg is not set")
		dbFile = ":memory:"
	}

	// Open the data.db file. It will be created if it doesn't exist.
	db, err := buntdb.Open(dbFile)
	if err != nil {
		log.Fatal(err)
	}

	//Set DB instance
	capi.DBInstance = db

	srv := Server{
		eventBus: eventBus,
		rpcBus:   rpcBus,
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
	capi.StartAPI(s.eventBus, s.rpcBus)

	r.HandleFunc("/consensus/bidders", capi.GetBidders).Methods("GET")
	r.HandleFunc("/consensus/provisioners", capi.GetProvisioners).Methods("GET")
	r.HandleFunc("/consensus/roundinfo", capi.GetRoundInfo).Methods("GET")
	r.HandleFunc("/consensus/eventqueuestatus", capi.GetEventQueueStatus).Methods("GET")

	return r
}