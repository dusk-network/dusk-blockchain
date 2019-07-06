package transactions

import (
	"math/big"
	"math/rand"
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/key"

	"github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
)

func TestAddOutputs(t *testing.T) {
	tx, netPrefix, _ := randomStandardTx(t)

	Alice := key.NewKeyPair([]byte("this is the users seed"))
	pubAddr, err := Alice.PublicKey().PublicAddress(netPrefix)
	assert.Nil(t, err)

	var amountToSend ristretto.Scalar
	amountToSend.SetBigInt(big.NewInt(20))

	for i := 0; i < maxOutputs; i++ {
		err = tx.AddOutput(*pubAddr, amountToSend)
		assert.Nil(t, err)
	}

	// One more should result in an error
	err = tx.AddOutput(*pubAddr, amountToSend)
	assert.NotNil(t, err)

	// TotalSent = maxOutputs * amountToSend
	assert.Equal(t, tx.TotalSent.BigInt().Int64(), maxOutputs*amountToSend.BigInt().Int64())
}

func TestAddMaxInputs(t *testing.T) {
	tx, _, _ := randomStandardTx(t)

	for i := 0; i < maxInputs; i++ {
		err := tx.AddInput(&Input{})
		assert.Nil(t, err)
	}

	err := tx.AddInput(&Input{})
	assert.NotNil(t, err)
}

func TestCalCommToZero(t *testing.T) {
	var amount, r ristretto.Scalar
	amount.SetBigInt(big.NewInt(rand.Int63()))
	r.Rand()

	var numInputs, numOutputs = 3, 3

	inputs := make([]*Input, 0, numInputs)
	for i := 0; i < numInputs; i++ {
		var mask ristretto.Scalar
		mask.Rand()
		inputs = append(inputs, NewInput(amount, mask, amount))
	}

	Alice := key.NewKeyPair([]byte("this is the users seed"))

	outputs := make([]*Output, 0, numOutputs)
	for i := 0; i < numOutputs; i++ {

		var mask ristretto.Scalar
		mask.Rand()

		output := NewOutput(r, amount, 0, *Alice.PublicKey())
		output.mask = mask
		output.Commitment = CommitAmount(output.amount, output.mask)

		outputs = append(outputs, output)
	}

	calculateCommToZero(inputs, outputs)

	var x, y ristretto.Point
	x.SetZero()
	y.SetZero()

	for i := range inputs {
		x.Add(&x, &inputs[i].PseudoCommitment)
	}
	for i := range outputs {
		y.Add(&y, &outputs[i].Commitment)
	}
	x.Sub(&x, &y) // Should be zero; proves that sumOuts - sumIns = 0

	var zero ristretto.Point
	zero.SetZero()
	assert.True(t, x.Equals(&zero))
}

func TestProve(t *testing.T) {
	tx, netPrefix, _ := randomStandardTx(t)

	// Add three Inputs to transaction
	addValueInputToTx(10, tx)
	addValueInputToTx(20, tx)
	addValueInputToTx(30, tx)
	assert.Equal(t, 3, len(tx.Inputs))

	// Add Decoys to Inputs
	err := tx.AddDecoys(11, generateDecoys)
	assert.Nil(t, err)

	// Add two outputs to transaction -- Must add upto input amounts
	addValueOutputToTx(t, 30, netPrefix, tx)
	addValueOutputToTx(t, 30, netPrefix, tx)

	// Create all necessary proofs
	err = tx.Prove()
	assert.Nil(t, err)
}

func TestAddDecoys(t *testing.T) {
	tx, _, _ := randomStandardTx(t)

	numInputs := 5
	numDecoys := 4

	for i := 0; i < numInputs; i++ {
		input := NewInput(ristretto.Scalar{}, ristretto.Scalar{}, ristretto.Scalar{})
		err := tx.AddInput(input)
		assert.Nil(t, err)
	}
	assert.Equal(t, numInputs, len(tx.Inputs))

	err := tx.AddDecoys(numDecoys, generateDecoys)
	assert.Nil(t, err)

	for i := range tx.Inputs {
		input := tx.Inputs[i]
		assert.Equal(t, numDecoys, input.Proof.LenMembers())
	}
}

func TestProveRangeProof(t *testing.T) {
	tx, netPrefix, _ := randomStandardTx(t)

	Alice := key.NewKeyPair([]byte("this is the users seed"))
	pubAddr, err := Alice.PublicKey().PublicAddress(netPrefix)
	assert.Nil(t, err)

	var amountToSend ristretto.Scalar
	amountToSend.SetBigInt(big.NewInt(rand.Int63()))

	for i := 0; i < maxOutputs; i++ {
		err = tx.AddOutput(*pubAddr, amountToSend)
		assert.Nil(t, err)
	}

	err = tx.ProveRangeProof()
	assert.Nil(t, err)

	// Check that each output now has the correct commitment
	for i := 0; i < maxOutputs; i++ {
		output := tx.Outputs[i]
		expected := CommitAmount(output.amount, output.mask)
		assert.True(t, output.Commitment.Equals(&expected))
	}
}

func randomStandardTx(t *testing.T) (*StandardTx, byte, int64) {
	r8 := rand.Uint32() >> 16

	netPrefix := byte(r8)
	fee := rand.Int63()

	tx, err := NewStandard(netPrefix, fee)
	assert.Nil(t, err)

	return tx, netPrefix, fee
}

func addValueInputToTx(value int64, tx *StandardTx) {
	amount := int64ToScalar(value)
	var privKey, mask ristretto.Scalar
	privKey.Rand()
	mask.Rand()
	tx.AddInput(NewInput(amount, mask, privKey))
}

func addValueOutputToTx(t *testing.T, value int64, netPrefix byte, tx *StandardTx) {
	amount := int64ToScalar(value)

	Alice := key.NewKeyPair([]byte("this is the users seed"))
	pubAddr, err := Alice.PublicKey().PublicAddress(netPrefix)
	assert.Nil(t, err)

	err = tx.AddOutput(*pubAddr, amount)
	assert.Nil(t, err)

}

func int64ToScalar(n int64) ristretto.Scalar {
	var x ristretto.Scalar
	x.SetBigInt(big.NewInt(n))
	return x
}

func randomSlice(n int) []byte {
	slice := make([]byte, n)
	rand.Read(slice)
	return slice
}
