// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package dupemap_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/stretchr/testify/assert"
)

func TestHas(t *testing.T) {
	testPayload := bytes.NewBufferString("This is a test")
	tmpMap := dupemap.NewTmpMap(1000, 5)

	assert.False(t, tmpMap.Has(testPayload))
	tmpMap.Add(testPayload)

	assert.True(t, tmpMap.Has(testPayload))
}

func TestAdd(t *testing.T) {
	testPayload := bytes.NewBufferString("This is a test")
	tmpMap := dupemap.NewTmpMap(1000, 5)

	assert.True(t, tmpMap.Add(testPayload))
}
