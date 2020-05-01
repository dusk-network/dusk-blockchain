package transactions

import (
	"fmt"
	"testing"

	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

func TestContractCallDecodeEncode(t *testing.T) {
	call := &rusk.ContractCallTx{
		ContractCall: &rusk.ContractCallTx_Tx{
			Tx: &rusk.Transaction{
				Inputs: []*rusk.TransactionInput{
					&rusk.TransactionInput{
						Note: &rusk.Note{
							NoteType:        0,
							Nonce:           &rusk.Nonce{Bs: []byte{0x11, 0x22}},
							RG:              &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
							PkR:             &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
							ValueCommitment: &rusk.Scalar{Data: []byte{0x55, 0x66}},
							BlindingFactor: &rusk.Note_TransparentBlindingFactor{
								TransparentBlindingFactor: &rusk.Scalar{Data: []byte{0x55, 0x66}},
							},
							Value: &rusk.Note_EncryptedValue{},
						},
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
					&rusk.TransactionOutput{
						Note: &rusk.Note{
							NoteType:        1,
							Nonce:           &rusk.Nonce{},
							RG:              &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
							PkR:             &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
							ValueCommitment: &rusk.Scalar{Data: []byte{0x55, 0x66}},
							BlindingFactor: &rusk.Note_TransparentBlindingFactor{
								TransparentBlindingFactor: &rusk.Scalar{Data: []byte{0x55, 0x66}},
							},
							Value: &rusk.Note_TransparentValue{},
						},
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
						BlindingFactor:  &rusk.Note_EncryptedBlindingFactor{},
						Value:           &rusk.Note_TransparentValue{},
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

	c, err := DecodeContractCall(call)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(c.(*Transaction))

	encodedCall, err := EncodeContractCall(c)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(encodedCall)
}
