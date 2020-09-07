package api

import (
	"bytes"
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"os"
	"testing"

	"github.com/drewolson/testflight"
	"github.com/stretchr/testify/require"
)

func TestConsensusAPISmokeTest(t *testing.T) {

	apiServer, err := NewHTTPServer(nil, nil)
	if err != nil {
		t.Errorf("API http server error: %v", err)
	}

	var tt = []struct {
		targetURL string
		name      string
		Data      string
	}{
		{
			targetURL: "/consensus/bidders?height=0",
			name:      "Get bidders",
			Data:      `{}`,
		},
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

	//setup viper timeout
	cwd, err := os.Getwd()
	require.Nil(t, err)

	r, err := cfg.LoadFromFile(cwd + "/../../dusk.toml")
	require.Nil(t, err)
	cfg.Mock(&r)

	apiServer, err := NewHTTPServer(nil, nil)
	require.Nil(t, err)

	provisioners, _ := consensus.MockProvisioners(5)

	err = apiServer.store.StoreProvisioners(provisioners, 1)
	require.Nil(t, err)

	provisioners, err = apiServer.store.FetchProvisioners(1)
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

	//setup viper timeout
	cwd, err := os.Getwd()
	require.Nil(t, err)

	r, err := cfg.LoadFromFile(cwd + "/../../dusk.toml")
	require.Nil(t, err)
	cfg.Mock(&r)

	apiServer, err := NewHTTPServer(nil, nil)
	require.Nil(t, err)

	for i := 0; i < 5; i++ {

		// steps array
		for j := 0; j < 5; j++ {
			err = apiServer.store.StoreRoundInfo(uint64(i), uint8(j), "StopConsensus", "")
			require.Nil(t, err)

			roundInfo, err := apiServer.store.FetchRoundInfo(uint64(i), 0, 5)
			require.Nil(t, err)
			require.NotNil(t, roundInfo)
		}

	}

	testflight.WithServer(apiServer.Server.Handler, func(r *testflight.Requester) {

		targetURL := "/consensus/roundinfo?height_begin=0&height_end=5"
		response := r.Get(targetURL)
		require.NotNil(t, response)

		require.NotEmpty(t, response.RawBody)
	})
}

func TestConsensusAPIEventStatus(t *testing.T) {

	//setup viper timeout
	cwd, err := os.Getwd()
	require.Nil(t, err)

	r, err := cfg.LoadFromFile(cwd + "/../../dusk.toml")
	require.Nil(t, err)
	cfg.Mock(&r)

	apiServer, err := NewHTTPServer(nil, nil)
	require.Nil(t, err)

	for i := 0; i < 5; i++ {

		// steps array
		for j := 0; j < 5; j++ {
			msg := message.New(topics.Initialization, bytes.Buffer{})
			err = apiServer.store.StoreEventQueue(uint64(i), uint8(j), msg)
			require.Nil(t, err)

			roundInfo, err := apiServer.store.FetchEventQueue(uint64(i), 0, 5)
			require.Nil(t, err)
			require.NotNil(t, roundInfo)
		}

	}

	testflight.WithServer(apiServer.Server.Handler, func(r *testflight.Requester) {

		targetURL := "/consensus/eventqueuestatus?height=1"
		response := r.Get(targetURL)
		require.NotNil(t, response)

		require.NotEmpty(t, response.RawBody)
	})
}
