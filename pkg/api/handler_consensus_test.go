package api_test

import (
	"testing"

	"github.com/drewolson/testflight"
	"github.com/dusk-network/dusk-blockchain/pkg/api"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/stretchr/testify/require"
)

func TestConsensusAPI(t *testing.T) {

	_, db := lite.CreateDBConnection()
	targetURL := "/consensus/provisioners"

	apiServer, err := api.NewHTTPServer(nil, nil, db)
	if err != nil {
		t.Errorf("API http server error: %v", err)
	}

	testflight.WithServer(apiServer.Server.Handler, func(r *testflight.Requester) {

		t.Log("testflight.WithServer --> ")

		response := r.Get(targetURL)
		require.NotNil(t, response)

	})
}
