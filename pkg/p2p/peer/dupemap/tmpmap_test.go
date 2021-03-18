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
	tmpMap := dupemap.NewTmpMap(3, 1000, 5)

	assert.False(t, tmpMap.Has(testPayload))

	tmpMap.AddAt(testPayload, 0)

	assert.True(t, tmpMap.Has(testPayload))

	tmpMap.UpdateHeight(2)
	assert.False(t, tmpMap.Has(testPayload))
	assert.False(t, tmpMap.HasAt(testPayload, 2))
	assert.True(t, tmpMap.HasAnywhere(testPayload))
	assert.True(t, tmpMap.HasAt(testPayload, uint64(0)))
}

func TestAdd(t *testing.T) {
	testPayload := bytes.NewBufferString("This is a test")
	tmpMap := dupemap.NewTmpMap(3, 1000, 5)

	assert.True(t, tmpMap.Add(testPayload))

	tmpMap.UpdateHeight(4)
	assert.True(t, tmpMap.Add(testPayload))
}

func TestClean(t *testing.T) {
	testPayload := bytes.NewBufferString("This is a test")
	tmpMap := dupemap.NewTmpMap(3, 1000, 5)

	assert.True(t, tmpMap.Add(testPayload))

	tmpMap.UpdateHeight(2)
	assert.True(t, tmpMap.Add(testPayload))

	tmpMap.UpdateHeight(5)
	assert.True(t, tmpMap.Add(testPayload))

	// this should clean entries at heigth 2 and less
	tmpMap.UpdateHeight(6)
	assert.True(t, tmpMap.Add(testPayload))

	assert.True(t, tmpMap.HasAnywhere(testPayload))
	assert.True(t, tmpMap.Has(testPayload))
	assert.False(t, tmpMap.HasAt(testPayload, 2))
	assert.True(t, tmpMap.HasAt(testPayload, 5))
}
