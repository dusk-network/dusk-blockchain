package transactions

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var tt = []struct {
	name string
	c    ContractCall
}{
	{
		"standard tx",
		RandTx(),
	},
	{
		"stake",
		RandStakeTx(0),
	},
	{
		"bid",
		RandBidTx(0),
	},
	{
		"coinbase",
		RandDistributeTx(400, 12),
	},
}

func TestContractCallDecodeEncode(t *testing.T) {
	assert := assert.New(t)
	for _, test := range tt {
		// we do not encode DistributeTransaction, so we skip it in this test
		if test.name == "coinbase" {
			continue
		}
		ruskTx, err := EncodeContractCall(test.c)
		if !assert.NoError(err) {
			t.Fatalf("encoding of %s failed: %v", test.name, err)
		}

		cc, err := DecodeContractCall(ruskTx)
		assert.NoError(err)
		if !assert.NoError(err) {
			t.Fatalf("decoding of %s failed: %v", test.name, err)
		}

		if !assert.Equal(test.c, cc) {
			t.Fatalf("equality violated after encoding/decoding of %s transaction", test.name)
		}
	}
}

func TestUnMarshal(t *testing.T) {
	assert := require.New(t)
	cc, _ := DecodeContractCall(RuskTx())
	assert.Equal(Tx, cc.Type())

	b := new(bytes.Buffer)
	err := Marshal(b, cc)
	assert.NoError(err)

	ccOther, uerr := Unmarshal(b)
	assert.NoError(uerr)

	assert.True(Equal(cc, ccOther))
}

func BenchmarkEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = DecodeContractCall(RuskTx())
	}
}

func BenchmarkDecode(b *testing.B) {
	c, _ := DecodeContractCall(RuskTx())
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = EncodeContractCall(c)
	}
}

/*
func encodeDecode(tx *rusk.ContractCallTx) error {
	c, err := DecodeContractCall(tx)
	if err != nil {
		return err
	}

	_, err = EncodeContractCall(c)
	if err != nil {
		return err
	}
	return nil
}
*/
