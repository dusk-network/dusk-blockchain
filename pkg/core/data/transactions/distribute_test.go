package transactions_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	assert "github.com/stretchr/testify/require"
)

func TestDistributeHashing(t *testing.T) {
	assert := assert.New(t)
	d := transactions.RandDistributeTx(50, 12)
	assert.NotPanics(func() { _, _ = d.CalculateHash() })
}

// This test Marshal and Unmarshal a DistributeTransaction using JSON. It tests
// the congruency of the marshaling with particular attention to the TxType
// which uses a custom JSON Marshaler
func TestJSONUnMarshaling(t *testing.T) {
	assert := assert.New(t)

	d := transactions.RandDistributeTx(50, 12)
	b, err := json.MarshalIndent(d, "", "  ")
	assert.NoError(err)

	test := &transactions.DistributeTransaction{}
	assert.NoError(json.Unmarshal(b, test))
	assert.Equal(d.BgPk, test.BgPk)
	h, err := d.CalculateHash()
	assert.NoError(err)

	htest, err := test.CalculateHash()
	assert.NoError(err)
	assert.Equal(h, htest)
}

func TestUnMashaling(t *testing.T) {
	assert := assert.New(t)
	d := transactions.RandDistributeTx(50, 12)

	b := new(bytes.Buffer)
	assert.NoError(transactions.MarshalDistribute(b, *d))
	assert.NoError(transactions.UnmarshalDistribute(b, d))
}

func TestCopy(t *testing.T) {
	assert := assert.New(t)
	d1 := transactions.RandDistributeTx(50, 12)
	d2 := d1.Copy().(*transactions.DistributeTransaction)
	b1 := new(bytes.Buffer)
	assert.NoError(transactions.MarshalDistribute(b1, *d1))
	b2 := new(bytes.Buffer)
	assert.NoError(transactions.MarshalDistribute(b2, *d2))
	assert.Equal(b1.Bytes(), b2.Bytes())
}
