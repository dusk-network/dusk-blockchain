// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package helper

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/common"
)

// const respAmount uint32 = 7

// RandomInput returns a random input for testing.
func RandomInput(t *testing.T) *common.BlsScalar {
	/*
		amount := ristretto.Scalar{}
		amount.Rand()
		privKey := ristretto.Scalar{}
		privKey.Rand()
		mask := ristretto.Scalar{}
		mask.Rand()

		sigBuf := randomSignatureBuffer(t)

		in := transactions.NewInput(amount, privKey, mask)
		in.Signature = &mlsag.Signature{}
		if err := in.Signature.Decode(sigBuf, true); err != nil {
			t.Fatal(err)
		}

		in.KeyImage.Rand()
		return in
	*/
	return nil
}

// RandomInputs returns a slice of inputs of size `size` for testing.
func RandomInputs(t *testing.T, size int) []*common.BlsScalar {
	/*
		var ins transactions.Inputs

		for i := 0; i < size; i++ {
			in := RandomInput(t)
			assert.NotNil(t, in)
			ins = append(ins, in)
		}

		return ins
	*/
	return nil
}
