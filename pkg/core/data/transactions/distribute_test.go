package transactions_test

import (
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
