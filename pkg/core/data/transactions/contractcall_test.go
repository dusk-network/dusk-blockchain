package transactions

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
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

func BenchmarkJSON(b *testing.B) {
	tx := RandTx()
	t := new(Transaction)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf, _ := json.Marshal(tx)
		_ = json.Unmarshal(buf, t)
	}
}

func BenchmarkWire(b *testing.B) {
	tx := RandTx()
	t := new(Transaction)
	buf := new(bytes.Buffer)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = MarshalTransaction(buf, *tx)
		_ = UnmarshalTransaction(buf, t)
	}
}

func BenchmarkCopy(b *testing.B) {
	tx := RandTx()
	t := new(Transaction)
	r := new(rusk.Transaction)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = MTx(r, tx)
		_ = UTx(r, t)
	}
}

func BenchmarkProto(b *testing.B) {
	tx := RandTx()
	r, _ := EncodeContractCall(tx)
	unr := new(rusk.ContractCallTx)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf, _ := proto.Marshal(r)
		_ = proto.Unmarshal(buf, unr)
	}
}
