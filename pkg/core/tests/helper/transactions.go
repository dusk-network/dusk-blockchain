package helper

import (
	"bytes"
	"encoding/binary"
	"io"
	"math/big"
	"testing"

	"github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-crypto/mlsag"
	"github.com/dusk-network/dusk-crypto/rangeproof"
	"github.com/dusk-network/dusk-wallet/v2/key"
	"github.com/dusk-network/dusk-wallet/v2/transactions"
	"github.com/stretchr/testify/assert"
)

const (
	lockTime = uint64(2000000000)
	fee      = int64(200)
)

var numInputs, numOutputs = 23, 16

// RandomSliceOfTxs returns a random slice of transactions for testing
// Each tx batch represents all 4 non-coinbase tx types
func RandomSliceOfTxs(t *testing.T, txsBatchCount uint16) []transactions.Transaction {
	var txs []transactions.Transaction

	txs = append(txs, RandomCoinBaseTx(t, false))

	var i uint16
	for ; i < txsBatchCount; i++ {

		txs = append(txs, RandomStandardTx(t, false))
		txs = append(txs, RandomTLockTx(t, false))

		stake, err := RandomStakeTx(t, false)
		assert.Nil(t, err)
		txs = append(txs, stake)

		bid, err := RandomBidTx(t, false)
		assert.Nil(t, err)
		txs = append(txs, bid)
	}

	return txs
}

// RandomBidTx returns a random bid transaction for testing
func RandomBidTx(t *testing.T, malformed bool) (*transactions.Bid, error) {
	var M = RandomSlice(t, 32)

	if malformed {
		M = RandomSlice(t, 12)
	}

	tx, err := transactions.NewBid(0, 2, fee, lockTime, M)
	if err != nil {
		return tx, err
	}
	rp := randomRangeProofBuffer(t)
	_ = tx.RangeProof.Decode(rp, true)

	// Inputs
	tx.Inputs = RandomInputs(t, numInputs)

	// Outputs
	tx.Outputs = RandomOutputs(t, numOutputs)

	// Set TxID
	hash, err := tx.CalculateHash()
	if err != nil {
		t.Fatal(err)
	}

	tx.TxID = hash

	return tx, err
}

// RandomCoinBaseTx returns a random coinbase transaction for testing
func RandomCoinBaseTx(t *testing.T, malformed bool) *transactions.Coinbase {
	proof := RandomSlice(t, 2000)
	score := RandomSlice(t, 32)

	tx := transactions.NewCoinbase(proof, score, 2)
	tx.Rewards = RandomOutputs(t, 1)
	tx.Rewards[0].EncryptedAmount.SetBigInt(big.NewInt(int64(config.GeneratorReward)))

	// Set TxID
	hash, err := tx.CalculateHash()
	if err != nil {
		t.Fatal(err)
	}

	tx.TxID = hash
	return tx
}

// RandomTLockTx returns a random timelock transaction for testing
func RandomTLockTx(t *testing.T, malformed bool) *transactions.Timelock {
	tx, err := transactions.NewTimelock(0, 2, fee, lockTime)
	if err != nil {
		t.Fatal(err)
	}
	rp := randomRangeProofBuffer(t)
	_ = tx.RangeProof.Decode(rp, true)

	// Inputs
	tx.Inputs = RandomInputs(t, numInputs)

	// Outputs
	tx.Outputs = RandomOutputs(t, numOutputs)

	// Set TxID
	hash, err := tx.CalculateHash()
	if err != nil {
		t.Fatal(err)
	}

	tx.TxID = hash

	return tx
}

// RandomStandardTx returns a random standard tx for testing
func RandomStandardTx(t *testing.T, malformed bool) *transactions.Standard {
	tx, err := transactions.NewStandard(0, 2, fee)
	if err != nil {
		t.Fatal(err)
	}
	rp := randomRangeProofBuffer(t)
	if e := tx.RangeProof.Decode(rp, true); e != nil {
		t.Fatal(e)
	}

	// Inputs
	tx.Inputs = RandomInputs(t, numInputs)

	// Outputs
	tx.Outputs = RandomOutputs(t, numOutputs)

	// Set TxID
	hash, err := tx.CalculateHash()
	if err != nil {
		t.Fatal(err)
	}

	tx.TxID = hash

	return tx
}

// RandomStakeTx returns a random stake tx for testing
func RandomStakeTx(t *testing.T, malformed bool) (*transactions.Stake, error) {
	edKey := RandomSlice(t, 32)
	blsKey := RandomSlice(t, 33)

	tx, err := transactions.NewStake(0, 2, fee, lockTime, edKey, blsKey)
	if err != nil {
		return tx, err
	}
	rp := randomRangeProofBuffer(t)
	_ = tx.RangeProof.Decode(rp, true)

	// Inputs
	tx.Inputs = RandomInputs(t, numInputs)

	// Outputs
	tx.Outputs = RandomOutputs(t, numOutputs)

	// Set TxID
	hash, err := tx.CalculateHash()
	if err != nil {
		t.Fatal(err)
	}

	tx.TxID = hash

	return tx, nil
}

//func fetchDecoys(numMixins int) []mlsag.PubKeys {
//	var decoys []ristretto.Point
//	for i := 0; i < numMixins; i++ {
//		decoy := ristretto.Point{}
//		decoy.Rand()
//
//		decoys = append(decoys, decoy)
//	}
//
//	var pubKeys []mlsag.PubKeys
//	for i := 0; i < numMixins; i++ {
//		var keyVector mlsag.PubKeys
//		keyVector.AddPubKey(decoys[i])
//
//		var secondaryKey ristretto.Point
//		secondaryKey.Rand()
//		keyVector.AddPubKey(secondaryKey)
//
//		pubKeys = append(pubKeys, keyVector)
//	}
//	return pubKeys
//}

func randomRangeProofBuffer(t *testing.T) *bytes.Buffer {
	lenComm := uint32(1)
	commBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(commBytes, lenComm)
	buf := bytes.NewBuffer(commBytes)
	comm := ristretto.Point{}
	comm.Rand()
	if _, err := buf.Write(comm.Bytes()); err != nil {
		t.Fatal(err)
	}

	// Create random points
	for i := 0; i < 4; i++ {
		writeRandomPoint(t, buf)
	}

	// Create random scalars
	for i := 0; i < 3; i++ {
		writeRandomScalar(t, buf)
	}

	writeRandomIPProof(t, buf)
	return buf
}

func writeRandomIPProof(t *testing.T, w io.Writer) {
	// Add scalars
	for i := 0; i < 2; i++ {
		writeRandomScalar(t, w)
	}

	// Add points
	for i := 0; i < 2; i++ {
		writeRandomPoint(t, w)
	}
}

func writeRandomScalar(t *testing.T, w io.Writer) {
	s := ristretto.Scalar{}
	s.Rand()
	if _, err := w.Write(s.Bytes()); err != nil {
		t.Fatal(err)
	}
}

func writeRandomPoint(t *testing.T, w io.Writer) {
	p := ristretto.Point{}
	p.Rand()
	if _, err := w.Write(p.Bytes()); err != nil {
		t.Fatal(err)
	}
}

func fixedRangeProof(t *testing.T) rangeproof.Proof {
	lenComm := uint32(1)
	commBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(commBytes, lenComm)
	buf := bytes.NewBuffer(commBytes)
	comm := ristretto.Point{}
	comm.SetZero()
	if _, err := buf.Write(comm.Bytes()); err != nil {
		t.Fatal(err)
	}

	// Create random points
	for i := 0; i < 4; i++ {
		p := ristretto.Point{}
		p.SetZero()
		if _, err := buf.Write(p.Bytes()); err != nil {
			t.Fatal(err)
		}
	}

	// Create random scalars
	for i := 0; i < 5; i++ {
		s := ristretto.Scalar{}
		s.SetZero()
		if _, err := buf.Write(s.Bytes()); err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < 2; i++ {
		p := ristretto.Point{}
		p.SetZero()
		if _, err := buf.Write(p.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
	rp := rangeproof.Proof{}
	if err := rp.Decode(buf, true); err != nil {
		t.Fatal(err)
	}

	return rp
}

// FixedStandardTx generates an encodable standard Tx with 1 input and 1 output
// It guarantees that for one seed the same standard Tx (incl. TxID) is
// always generated.
func FixedStandardTx(t *testing.T, seed uint64) transactions.Transaction {

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
}
