// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package server_test

import (
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestCreateDropSession(t *testing.T) {
	t.Skip()
	assert := assert.New(t)
	jwt, err := authClient.CreateSession()
	assert.NoError(err)
	assert.NotEmpty(jwt)

	// first time we drop the session there should be no error
	assert.NoError(authClient.DropSession())
	// if we drop the session immediately after, we should get an
	// authorization error since the session has been dropped and we did not
	// recreate it
	assert.Error(authClient.DropSession())
}
