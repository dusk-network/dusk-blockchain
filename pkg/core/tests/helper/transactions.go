package helper

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
)

// FixedStandardTx generates an encodable standard Tx with 1 input and 1 output
// It guarantees that for one seed the same standard Tx (incl. TxID) is
// always generated.
// FIXME: 458 - this is heavily used by the gql package
func FixedStandardTx(t *testing.T, seed uint64) transactions.ContractCall {
	/*
		seedScalar := ristretto.Scalar{}
		seedScalar.SetBigInt(big.NewInt(0).SetUint64(seed))

		seedPoint := ristretto.Point{}
		seedPoint.ScalarMultBase(&seedScalar)

		netPrefix := byte(2)
		tx, err := transactions.NewStandard(0, netPrefix, int64(seed))
		if err != nil {
			t.Fatal(err)
		}

		tx.R = seedPoint
		tx.RangeProof = fixedRangeProof(t)

		// Add a fixed output
		walletKeys := key.NewKeyPair([]byte{5, 0, 0})
		output := transactions.NewOutput(seedScalar, seedScalar, 0, *walletKeys.PublicKey())
		tx.Outputs = append(tx.Outputs, output)

		privKey, _ := walletKeys.PrivateSpend()
		in := transactions.NewInput(seedScalar, seedScalar, ristretto.Scalar(*privKey))

		sigBuf := fixedSignatureBuffer(t)
		in.Signature = &mlsag.Signature{}
		if err := in.Signature.Decode(sigBuf, true); err != nil {
			t.Fatal(err)
		}

		in.KeyImage = seedPoint
		tx.Inputs = append(tx.Inputs, in)

		return tx
	*/
	return nil
}
