package transactions

import (
	"testing"

	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	assert "github.com/stretchr/testify/require"
)

func TestContractCallDecodeEncode(t *testing.T) {
	assert.NoError(t, encodeDecode(RuskTx))
}

func TestUnMarshal(t *testing.T) {
	assert := assert.New(t)
	cc, _ := DecodeContractCall(RuskTx)
	assert.Equal(Tx, cc.Type())

	b, err := Marshal(cc)
	assert.NoError(err)

	ccOther := new(Transaction)
	assert.NoError(Unmarshal(b, ccOther))

	assert.True(Equal(cc, ccOther))
}

func BenchmarkEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = DecodeContractCall(RuskTx)
	}
}

func BenchmarkDecode(b *testing.B) {
	c, _ := DecodeContractCall(RuskTx)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = EncodeContractCall(c)
	}
}

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
