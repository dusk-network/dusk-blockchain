package capi

import (
	"net/http"
	"strconv"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	log "github.com/sirupsen/logrus"
)

var (
	eventBus *eventbus.EventBus
	rpcBus   *rpcbus.RPCBus
	memoryDB database.DB
)

func StartAPI(eb *eventbus.EventBus, rb *rpcbus.RPCBus, db database.DB) {
	eventBus = eb
	rpcBus = rb
	memoryDB = db
}

func GetBidders(res http.ResponseWriter, req *http.Request) {
	heightStr := req.URL.Query().Get(":height")
	if heightStr == "" {
		res.WriteHeader(http.StatusBadRequest)
	}
	height, err := strconv.Atoi(heightStr)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
	}

	log.WithField("height", height).Debug("GetBidders")
	_, _ = res.Write([]byte(``))

	res.WriteHeader(http.StatusOK)
}

func GetProvisioners(res http.ResponseWriter, req *http.Request) {
	heightStr := req.URL.Query().Get(":height")
	if heightStr == "" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	height, err := strconv.Atoi(heightStr)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	log.WithField("height", height).Debug("GetProvisioners")

	var provisioners []byte
	err = memoryDB.View(func(t database.Transaction) error {
		var err1 error
		provisioners, err1 = t.FetchProvisioners(uint64(height))
		if err1 != nil {
			return err1
		}
		return nil
	})

	if err != nil {
		res.WriteHeader(http.StatusNotFound)
		return
	}

	_, _ = res.Write(provisioners)

	res.WriteHeader(http.StatusOK)
}

func GetCurrentStep(res http.ResponseWriter, req *http.Request) {

	res.WriteHeader(http.StatusOK)
}

func GetEventQueueStatus(res http.ResponseWriter, req *http.Request) {

	res.WriteHeader(http.StatusOK)
}
