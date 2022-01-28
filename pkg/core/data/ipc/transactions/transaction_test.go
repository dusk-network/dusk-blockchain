// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package transactions

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"github.com/stretchr/testify/assert"
)

func equal(assert *assert.Assertions, tx1, tx2 *Transaction) {
	assert.Equal(tx1.TxType, tx2.TxType)
	assert.Equal(tx1.Version, tx2.Version)
	assert.True(bytes.Equal(tx1.Payload.Data, tx2.Payload.Data))

	assert.Equal(tx1.FeeValue.GasLimit, tx2.FeeValue.GasLimit)
	assert.Equal(tx1.FeeValue.GasPrice, tx2.FeeValue.GasPrice)
}

func TestMarshal(t *testing.T) {
	assert := assert.New(t)

	tx := RandTx()

	var buf bytes.Buffer
	err := Marshal(&buf, tx)
	assert.NoError(err)

	unmarshaled := NewTransaction()
	err = Unmarshal(&buf, unmarshaled)
	assert.NoError(err)

	equal(assert, tx, unmarshaled)
}

func TestCopy(t *testing.T) {
	assert := assert.New(t)

	tx := RandTx()
	cpy := tx.Copy()

	equal(assert, tx, cpy.(*Transaction))
}

func TestCalculateHash(t *testing.T) {
	assert := assert.New(t)

	tx := RandTx()

	hash, _ := tx.CalculateHash()
	assert.True(bytes.Equal(tx.Hash[:], hash))
}

func TestConversion(t *testing.T) {
	assert := assert.New(t)

	tx1 := RandTx()

	var r rusk.Transaction
	MTransaction(&r, tx1)

	tx2 := NewTransaction()
	UTransaction(&r, tx2)

	assert.Equal(tx1.TxType, tx2.TxType)
	assert.Equal(tx1.Version, tx2.Version)
	assert.True(bytes.Equal(tx1.Payload.Data, tx2.Payload.Data))
}
