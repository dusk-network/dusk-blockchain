package api_test

import (
	"testing"

	"github.com/drewolson/testflight"
	"github.com/dusk-network/dusk-blockchain/pkg/api"
	"github.com/stretchr/testify/require"
)

func TestConsensusAPI(t *testing.T) {

	apiServer, err := api.NewHTTPServer(nil, nil)
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

				//TODO: implement prepare dataset before query

				response := r.Get(tc.targetURL)
				require.NotNil(t, response)

				// TODO: assert response
				//require.Equal(t, 200, response.StatusCode)
			})
		}

	})
}
