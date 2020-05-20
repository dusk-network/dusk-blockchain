package transactions

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var tt = []struct {
	c   ContractCall
	typ reflect.Type
}{
	{
		RandTx(),
		reflect.TypeOf((*Transaction)(nil)),
	},
	{
		RandStakeTx(0),
		reflect.TypeOf((*StakeTransaction)(nil)),
	},
	{
		RandBidTx(0),
		reflect.TypeOf((*BidTransaction)(nil)),
	},
	{
		RandDistributeTx(400, 12),
		reflect.TypeOf((*DistributeTransaction)(nil)),
	},
}

func TestContractCallDecodeEncode(t *testing.T) {
	assert := assert.New(t)
	for _, test := range tt {
		// we do not encode DistributeTransaction, so we skip it in this test
		if test.typ.String() == "*transactions.DistributeTransaction" {
			continue
		}
		ruskTx, err := EncodeContractCall(test.c)
		if !assert.NoError(err) {
			t.Fatalf("encoding of %s failed: %v", test.typ.Name(), err)
		}

		cc, err := DecodeContractCall(ruskTx)
		assert.NoError(err)
		if !assert.NoError(err) {
			t.Fatalf("decoding of %s failed: %v", test.typ.Name(), err)
		}

		if !assert.Equal(test.c, cc) {
			t.Fatalf("equality violated after encoding/decoding of %s transaction", test.typ.Name())
		}
	}
}

func TestContractCallUnMarshal(t *testing.T) {
	assert := assert.New(t)
	for _, test := range tt {
		b := new(bytes.Buffer)

		if !assert.NoError(Marshal(b, test.c)) {
			t.Fatalf("marshaling of %s failed", test.typ.Name())
		}

		cc, err := Unmarshal(b)
		if !assert.NoError(err) {
			t.Fatalf("unmarshaling of %s failed", test.typ.Name())
		}

		if !assert.Equal(test.c, cc) {
			t.Fatalf("equality violated after marshaling/unmarshaling of %s transaction", test.typ.Name())
		}
	}
}

func TestContractCallJSONUnMarshal(t *testing.T) {
	assert := assert.New(t)
	for _, test := range tt {
		b, err := json.Marshal(test.c)
		if !assert.NoError(err) {
			t.Fatalf("marshaling of %s failed", test.typ.Name())
		}

		cc := reflect.New(test.typ.Elem())
		ccIntrf := cc.Interface()
		if !assert.NoError(json.Unmarshal(b, ccIntrf)) { //nolint
			t.Fatalf("unmarshaling of %s failed", test.typ.Name())
		}

		if !assert.Equal(test.c, ccIntrf) {
			t.Fatalf("equality violated after marshaling/unmarshaling of %s transaction", test.typ.Name())
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
