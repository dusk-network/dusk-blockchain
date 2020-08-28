package api_test

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"testing"

	"github.com/drewolson/testflight"
	"github.com/dusk-network/dusk-blockchain/pkg/api"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/stretchr/testify/require"
)

func TestConsensusAPI(t *testing.T) {

	_, dbheavy := heavy.CreateDBConnection()
	_, db := lite.CreateDBConnection()

	apiServer, err := api.NewHTTPServer(nil, nil, dbheavy, db)
	if err != nil {
		t.Errorf("API http server error: %v", err)
	}

	var tt = []struct {
		targetURL     string
		name          string
		ExpectSuccess bool
	}{
		{
			targetURL:     "/consensus/provisioners",
			name:          "Get provisioners",
			ExpectSuccess: true,
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
