package capi

import (
	"net/http"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

var (
	eventBus *eventbus.EventBus
	rpcBus   *rpcbus.RPCBus
	db       database.DB
)

func StartAPI(eb *eventbus.EventBus, rb *rpcbus.RPCBus, thisdb database.DB) {
	eventBus = eb
	rpcBus = rb
	db = thisdb
}

func GetBidders(res http.ResponseWriter, req *http.Request) {
	height := req.URL.Query().Get(":height")

	if height == "" {

	}

	res.WriteHeader(http.StatusOK)
}

func GetProvisioners(res http.ResponseWriter, req *http.Request) {
	height := req.URL.Query().Get(":height")

	if height == "" {

	}

	res.WriteHeader(http.StatusOK)
}

func GetCurrentStep(res http.ResponseWriter, req *http.Request) {

	res.WriteHeader(http.StatusOK)
}

func GetEventQueueStatus(res http.ResponseWriter, req *http.Request) {

	res.WriteHeader(http.StatusOK)
}
