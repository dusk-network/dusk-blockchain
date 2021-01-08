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

var dupeTests = []struct {
	height    uint64
	tolerance uint64
	canFwd    bool
}{
	{1, 3, true},
	{1, 3, false},
	{2, 3, false},
	{4, 3, true},
	{4, 3, false},
	{5, 3, false},
	{7, 1, true},
	{8, 1, false},
	{9, 1, true},
}

func TestDupeFilter(t *testing.T) {
	dupeMap := dupemap.NewDupeMap(1)
	test := bytes.NewBufferString("This is a test")
	for _, tt := range dupeTests {
		dupeMap.UpdateHeight(tt.height)
		dupeMap.SetTolerance(tt.tolerance)
		res := dupeMap.CanFwd(test)
		if !assert.Equal(t, tt.canFwd, res) {
			assert.FailNowf(t, "failure", "DupeMap.CanFwd: expected %t, got %t with height %d and tolerance %d", res, tt.canFwd, tt.height, tt.tolerance)
		}
	}
}
