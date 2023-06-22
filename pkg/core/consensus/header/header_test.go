// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package header_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	assert "github.com/stretchr/testify/require"
)

// Test that the Header is a functional payload.Safe implementation.
func TestCopy(t *testing.T) {
	assert := assert.New(t)
	red := header.Header{}
	red.Round = uint64(1)
	red.Step = uint8(2)
	red.BlockHash, _ = crypto.RandEntropy(32)
	red.PubKeyBLS, _ = crypto.RandEntropy(96)

	h := red.Copy().(header.Header)
	assert.True(reflect.DeepEqual(red, h))
}

// Test that the MarshalSignableVote and UnmarshalSignableVote functions store/retrieve
// the passed data properly.
func TestUnMarshalFields(t *testing.T) {
	assert := assert.New(t)
	red := header.Header{}
	red.Round = uint64(1)
	red.Step = uint8(2)
	red.BlockHash, _ = crypto.RandEntropy(32)
	test := header.Header{}
	buf := new(bytes.Buffer)
	assert.NoError(header.MarshalFields(buf, red))
	assert.NoError(header.UnmarshalFields(buf, &test))
	assert.Equal(red, test)
}

// This test ensures that both the Marshal and Unmarshal functions work as expected.
// The data which is stored and retrieved by both functions should remain the same.
func TestUnMarshal(t *testing.T) {
	assert := assert.New(t)
	header1 := header.Header{}
	header1.PubKeyBLS, _ = crypto.RandEntropy(129)
	header1.Round = uint64(5)
	header1.Step = uint8(2)
	header1.BlockHash, _ = crypto.RandEntropy(32)

	buf := new(bytes.Buffer)
	assert.NoError(header.Marshal(buf, header1))

	header2 := header.Header{}
	assert.NoError(header.Unmarshal(buf, &header2))

	assert.True(header1.Equal(header2))
}
