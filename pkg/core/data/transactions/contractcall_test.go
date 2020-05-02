package transactions

import (
	"testing"

	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"github.com/stretchr/testify/assert"
)

func TestContractCallDecodeEncode(t *testing.T) {
	callTx := mockCall()
	assert.NoError(t, encodeDecode(callTx))
}

func TestUnMarshal(t *testing.T) {
	callTx := mockCall()
	cc, _ := DecodeContractCall(callTx)
	assert.Equal(t, Tx, cc.Type())

	b, err := Marshal(cc)
	assert.NoError(t, err)

	ccOther := new(Transaction)
	assert.NoError(t, Unmarshal(b, ccOther))

	assert.True(t, Equal(cc, ccOther))
}

func BenchmarkEncode(b *testing.B) {
	callTx := mockCall()

	for i := 0; i < b.N; i++ {
		_, _ = DecodeContractCall(callTx)
	}
}

func BenchmarkDecode(b *testing.B) {
	callTx := mockCall()
	c, _ := DecodeContractCall(callTx)
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

func mockCall() *rusk.ContractCallTx {
	return &rusk.ContractCallTx{
		ContractCall: &rusk.ContractCallTx_Tx{
			Tx: &rusk.Transaction{
				Inputs: []*rusk.TransactionInput{
					{
						Note: mockNote1(),
						Sk: &rusk.SecretKey{
							A: &rusk.Scalar{Data: []byte{0x55, 0x66}},
							B: &rusk.Scalar{Data: []byte{0x55, 0x66}},
						},
						Nullifier: &rusk.Nullifier{
							H: &rusk.Scalar{Data: []byte{0x55, 0x66}},
						},
						MerkleRoot: &rusk.Scalar{Data: []byte{0x55, 0x66}},
					},
				},
				Outputs: []*rusk.TransactionOutput{
					{
						Note: mockNote2(),
						Pk: &rusk.PublicKey{
							AG: &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
							BG: &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
						},
						BlindingFactor: &rusk.Scalar{Data: []byte{0x55, 0x66}},
					},
				},
				Fee: &rusk.TransactionOutput{
					Note: &rusk.Note{
						NoteType:        0,
						Nonce:           &rusk.Nonce{},
						RG:              &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
						PkR:             &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
						ValueCommitment: &rusk.Scalar{Data: []byte{0x55, 0x66}},
						BlindingFactor: &rusk.Note_TransparentBlindingFactor{
							TransparentBlindingFactor: &rusk.Scalar{Data: []byte{0x56, 0x67}},
						},
						Value: &rusk.Note_TransparentValue{
							TransparentValue: uint64(200),
						},
					},
					Pk: &rusk.PublicKey{
						AG: &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
						BG: &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
					},
					BlindingFactor: &rusk.Scalar{Data: []byte{0x55, 0x66}},
				},
				Proof: []byte{0xaa, 0xbb},
			},
		},
	}
}
