// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package capi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/asdine/storm/v3/q"

	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/sirupsen/logrus"
)

var (
	eventBus *eventbus.EventBus
	rpcBus   *rpcbus.RPCBus
	log      = logrus.WithField("package", "capi")
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

	log.WithField("height", height).Debug("GetProvisionersHandler")
	var provisioner ProvisionerJSON
	err = GetStormDBInstance().Find("ID", uint64(height), &provisioner)
	if err != nil {
		res.WriteHeader(http.StatusNotFound)
		return
	}

	var b []byte
	b, err = json.Marshal(provisioner)
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
		Debug("GetRoundInfoHandler")

	var roundInfos []RoundInfoJSON

	//TODO: step should be a argument for query ?
	err = GetStormDBInstance().DB.Range("Round", heightBegin, heightEnd, &roundInfos)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
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

	log.WithField("height", height).Debug("GetEventQueueStatusHandler")

	//TODO: stepBegin and stepEnd should be req parameters ?
	var eventQueueList []EventQueueJSON
	err = GetStormDBInstance().DB.Select(q.Gte("Round", uint64(height)), q.Lte("Round", uint64(height))).Find(&eventQueueList)
	if err != nil {
		log.WithError(err).Error("could not execute query GetEventQueueStatusHandler")
		res.WriteHeader(http.StatusNotFound)
		return
	}

	var b []byte
	b, err = json.Marshal(eventQueueList)
	if err != nil {
		res.WriteHeader(http.StatusNotFound)
		return
	}

	_, _ = res.Write(b)
}

// GetP2PLogsHandler will return PeerJSON json
func GetP2PLogsHandler(res http.ResponseWriter, req *http.Request) {
	typeStr := req.URL.Query().Get("type")
	if typeStr == "" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	log.WithField("typeStr", typeStr).Debug("GetP2PLogsHandler")

	var peerList []PeerJSON
	err := GetStormDBInstance().DB.Find("Type", typeStr, &peerList)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	var b []byte
	b, err = json.Marshal(peerList)
	if err != nil {
		res.WriteHeader(http.StatusNotFound)
		return
	}

	_, _ = res.Write(b)
}

// GetP2PCountHandler will return the current peer count
func GetP2PCountHandler(res http.ResponseWriter, req *http.Request) {
	peersCount, err := GetStormDBInstance().DB.Count(&PeerCount{})
	if err != nil {
		log.WithError(err).Debug("failed to count peers")
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	count := Count{
		Count: peersCount,
	}

	var b []byte
	b, err = json.Marshal(count)
	if err != nil {
		res.WriteHeader(http.StatusNotFound)
		return
	}

	_, _ = res.Write(b)
}
