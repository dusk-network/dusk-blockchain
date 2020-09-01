package capi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"

	"github.com/tidwall/buntdb"

	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	log "github.com/sirupsen/logrus"
)

var (
	eventBus *eventbus.EventBus
	rpcBus   *rpcbus.RPCBus
	// DBInstance holds the instance to manipulate the API monitoring DB
	DBInstance *buntdb.DB
)

// StartAPI init consensus API pointers
func StartAPI(eb *eventbus.EventBus, rb *rpcbus.RPCBus) {
	eventBus = eb
	rpcBus = rb

	log.
		WithField("eventBus", eventBus).
		WithField("rpcBus", rpcBus).
		Debug("StartAPI")
}

// GetBiddersHandler will return a json response
//FIXME this is not yet implemented since we dont have the info yet
func GetBiddersHandler(res http.ResponseWriter, req *http.Request) {
	heightStr := req.URL.Query().Get("height")
	if heightStr == "" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	height, err := strconv.Atoi(heightStr)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	log.WithField("height", height).Debug("GetBidders")
	_, _ = res.Write([]byte(`{"error":"not yet implemented"}`))

}

// GetProvisionersHandler will return Provisioners json
func GetProvisionersHandler(res http.ResponseWriter, req *http.Request) {
	heightStr := req.URL.Query().Get("height")
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
	var provisioners *user.Provisioners
	provisioners, err = FetchProvisioners(uint64(height))
	if err != nil {
		res.WriteHeader(http.StatusNotFound)
		return
	}

	var b []byte
	b, err = json.Marshal(provisioners)
	if err != nil {
		res.WriteHeader(http.StatusNotFound)
		return
	}
	_, _ = res.Write(b)
}

// GetRoundInfoHandler will return RoundInfoJSON json array
func GetRoundInfoHandler(res http.ResponseWriter, req *http.Request) {
	heightBeginStr := req.URL.Query().Get("height_begin")
	if heightBeginStr == "" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	heightBegin, err := strconv.Atoi(heightBeginStr)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	heightEndStr := req.URL.Query().Get("height_end")
	if heightEndStr == "" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	heightEnd, err := strconv.Atoi(heightEndStr)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	log.
		WithField("heightBegin", heightBegin).
		WithField("heightEnd", heightEnd).
		Debug("GetRoundInfo")

	var roundInfos []RoundInfoJSON

	count := heightEnd - heightBegin
	for i := 0; i < count; i++ {
		roundInfo, err1 := FetchRoundInfo(uint64(heightBegin + i))
		if err1 != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		roundInfos = append(roundInfos, roundInfo)
	}

	if len(roundInfos) == 0 {
		res.WriteHeader(http.StatusNotFound)
		return
	}

	outputBytes, err := json.Marshal(roundInfos)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, _ = res.Write(outputBytes)
}

// GetEventQueueStatusHandler will return EventQueueJSON json
func GetEventQueueStatusHandler(res http.ResponseWriter, req *http.Request) {
	heightStr := req.URL.Query().Get("height")
	if heightStr == "" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	height, err := strconv.Atoi(heightStr)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	log.WithField("height", height).Debug("GetEventQueueStatus")

	provisioners, err := FetchEventQueue(uint64(height))
	if err != nil {
		res.WriteHeader(http.StatusNotFound)
		return
	}

	var b []byte
	b, err = json.Marshal(provisioners)
	if err != nil {
		res.WriteHeader(http.StatusNotFound)
		return
	}

	_, _ = res.Write(b)
}
