// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package api

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/asdine/storm/v3/q"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/capi"

	"github.com/drewolson/testflight"
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/stretchr/testify/require"
)

func TestConsensusAPISmokeTest(t *testing.T) {
	apiServer, err := NewHTTPServer(nil, nil)
	if err != nil {
		t.Errorf("API http server error: %v", err)
	}

	tt := []struct {
		targetURL string
		name      string
		Data      string
	}{
		{
			targetURL: "/consensus/provisioners?height=0",
			name:      "Get provisioners",
			Data:      `{}`,
		},
		{
			targetURL: "/consensus/roundinfo?height_begin=0&height_end=0",
			name:      "Get round info",
			Data:      `{}`,
		},
		{
			targetURL: "/consensus/eventqueuestatus",
			name:      "Get event queue status",
			Data:      `{}`,
		},
	}

	testflight.WithServer(apiServer.Server.Handler, func(r *testflight.Requester) {
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				response := r.Get(tc.targetURL)
				require.NotNil(t, response)
			})
		}
	})
}

func TestConsensusAPIProvisioners(t *testing.T) {
	// setup viper timeout
	cwd, err := os.Getwd()
	require.Nil(t, err)

	r, err := cfg.LoadFromFile(cwd + "/../../dusk.toml")
	require.Nil(t, err)
	cfg.Mock(&r)

	apiServer, err := NewHTTPServer(nil, nil)
	require.Nil(t, err)

	provisioners, _ := consensus.MockProvisioners(5)
	members := make([]*capi.Member, len(provisioners.Members))
	i := 0

	for _, v := range provisioners.Members {
		var stakes []capi.Stake

		for _, s := range v.Stakes {
			stake := capi.Stake{
				Amount:      s.Amount,
				StartHeight: s.StartHeight,
				EndHeight:   s.EndHeight,
			}
			stakes = append(stakes, stake)
		}

		member := capi.Member{
			PublicKeyBLS: v.PublicKeyBLS,
			Stakes:       stakes,
		}

		members[i] = &member
		i++
	}

	provisioner := capi.ProvisionerJSON{
		ID:      1,
		Set:     provisioners.Set,
		Members: members,
	}

	err = apiServer.store.Save(&provisioner)
	require.Nil(t, err)

	var provisionerJSON capi.ProvisionerJSON
	err = apiServer.store.Find("ID", uint64(1), &provisionerJSON)
	require.Nil(t, err)
	require.NotNil(t, provisioners)

	testflight.WithServer(apiServer.Server.Handler, func(r *testflight.Requester) {
		targetURL := "/consensus/provisioners?height=1"
		response := r.Get(targetURL)
		require.NotNil(t, response)

		require.NotEmpty(t, response.RawBody)
	})
}

func TestConsensusAPIRoundInfo(t *testing.T) {
	// setup viper timeout
	cwd, err := os.Getwd()
	require.Nil(t, err)

	r, err := cfg.LoadFromFile(cwd + "/../../dusk.toml")
	require.Nil(t, err)
	cfg.Mock(&r)

	apiServer, err := NewHTTPServer(nil, nil)
	require.Nil(t, err)

	for i := 1; i < 6; i++ {
		// steps array
		for j := 0; j < 5; j++ {
			roundInfo := capi.RoundInfoJSON{
				Round:  uint64(i),
				Step:   uint8(j),
				Method: "StopConsensus",
				Name:   "",
			}
			err = apiServer.store.Save(&roundInfo)
			require.Nil(t, err)

			var roundInfoArr []capi.RoundInfoJSON
			err := apiServer.store.DB.Select(q.Gte("ID", uint64(0)), q.Lte("ID", 5)).Find(&roundInfoArr)
			require.Nil(t, err)
			require.NotNil(t, roundInfo)
		}
	}

	testflight.WithServer(apiServer.Server.Handler, func(r *testflight.Requester) {
		for i := 0; i < 5; i++ {
			targetURL := fmt.Sprintf("/consensus/roundinfo?height_begin=%d&height_end=6", i)
			response := r.Get(targetURL)
			require.NotNil(t, response)

			require.NotEmpty(t, response.RawBody)

			require.True(t, len(response.RawBody) > 50)

			body := string(response.RawBody)
			fmt.Println("roundinfo body", body)
		}
	})
}

func TestConsensusAPIEventStatus(t *testing.T) {
	// setup viper timeout
	cwd, err := os.Getwd()
	require.Nil(t, err)

	r, err := cfg.LoadFromFile(cwd + "/../../dusk.toml")
	require.Nil(t, err)
	cfg.Mock(&r)

	apiServer, err := NewHTTPServer(nil, nil)
	require.Nil(t, err)

	for i := 1; i < 6; i++ {
		// steps array
		for j := 0; j < 5; j++ {
			eventQueue := capi.EventQueueJSON{
				Round:     uint64(i),
				Step:      uint8(j),
				UpdatedAt: time.Now(),
			}
			err = apiServer.store.Save(&eventQueue)
			require.Nil(t, err)

			var eventQueueList []capi.EventQueueJSON
			err := apiServer.store.DB.Select(q.Gte("Round", uint64(0)), q.Lte("Round", 5)).Find(&eventQueueList)
			require.Nil(t, err)
			require.NotNil(t, eventQueueList)
		}
	}

	testflight.WithServer(apiServer.Server.Handler, func(r *testflight.Requester) {
		for i := 1; i < 6; i++ {
			targetURL := fmt.Sprintf("/consensus/eventqueuestatus?height=%d", i)
			response := r.Get(targetURL)
			require.NotNil(t, response)
			require.NotEmpty(t, response.RawBody)

			require.True(t, len(response.Body) > 100)
			body := response.Body
			fmt.Println(body)
		}
	})
}

func TestP2PLogsReader(t *testing.T) {
	// setup viper timeout
	cwd, err := os.Getwd()
	require.Nil(t, err)

	r, err := cfg.LoadFromFile(cwd + "/../../dusk.toml")
	require.Nil(t, err)
	cfg.Mock(&r)

	apiServer, err := NewHTTPServer(nil, nil)
	require.Nil(t, err)

	// steps array
	for j := 0; j < 5; j++ {
		peerJSON := capi.PeerJSON{
			Address:  fmt.Sprintf("127.0.0.1:7485%d", j),
			Type:     "Reader",
			Method:   "Accept",
			LastSeen: time.Now(),
		}
		err = apiServer.store.Save(&peerJSON)
		require.Nil(t, err)

		var peerList []capi.PeerJSON
		err := apiServer.store.DB.Find("Type", "Reader", &peerList)
		require.Nil(t, err)
		require.NotNil(t, peerList)
	}

	testflight.WithServer(apiServer.Server.Handler, func(r *testflight.Requester) {
		targetURL := "/p2p/logs?type=Reader"
		response := r.Get(targetURL)
		require.NotNil(t, response)
		require.NotEmpty(t, response.RawBody)

		body := string(response.RawBody)
		fmt.Println(body)

		require.True(t, len(body) > 100)
	})
}

func TestP2PLogsWriter(t *testing.T) {
	// setup viper timeout
	cwd, err := os.Getwd()
	require.Nil(t, err)

	r, err := cfg.LoadFromFile(cwd + "/../../dusk.toml")
	require.Nil(t, err)
	cfg.Mock(&r)

	apiServer, err := NewHTTPServer(nil, nil)
	require.Nil(t, err)

	// steps array
	for j := 0; j < 5; j++ {
		peerJSON := capi.PeerJSON{
			Address:  fmt.Sprintf("127.0.0.1:7485%d", j),
			Type:     "Writer",
			Method:   "Accept",
			LastSeen: time.Now(),
		}
		err = apiServer.store.Save(&peerJSON)
		require.Nil(t, err)

		var peerList []capi.PeerJSON
		err := apiServer.store.DB.Find("Type", "Writer", &peerList)
		require.Nil(t, err)
		require.NotNil(t, peerList)
	}

	testflight.WithServer(apiServer.Server.Handler, func(r *testflight.Requester) {
		targetURL := "/p2p/logs?type=Writer"
		response := r.Get(targetURL)
		require.NotNil(t, response)
		require.NotEmpty(t, response.RawBody)

		body := string(response.RawBody)
		fmt.Println(body)

		require.True(t, len(body) > 100)
	})
}
