package transactions

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-crypto/mlsag"
	"github.com/dusk-network/dusk-crypto/rangeproof"
	"github.com/dusk-network/dusk-wallet/key"

	"github.com/bwesterb/go-ristretto"
)

const minDecoys = 7
const maxInputs = 2000
const maxOutputs = 16

type FetchDecoys func(numMixins int) []mlsag.PubKeys

// Standard is a generic transaction. It can also be seen as a stealth transaction.
// It is used to make basic payments on the dusk network.
type Standard struct {
	//// Encoded fields
	// TxType represents the transaction type
	TxType TxType

	// R is the transaction Public Key
	R ristretto.Point

	// Version is the transaction version. It does not use semver.
	// A new transaction version denotes a modification of the previous structure
	Version uint8 // 1 byte

	// Inputs represent a list of inputs to the transaction.
	Inputs
	// Outputs represent a list of outputs to the transaction
	Outputs

	Fee ristretto.Scalar

	// RangeProof is the bulletproof rangeproof that proves that the hidden amount
	// is between 0 and 2^64
	RangeProof rangeproof.Proof

	////
	//// Non-encoded fields
	r     ristretto.Scalar
	index uint32

	// Prefix that signifies for which network this transaction is intended (testnet, mainnet)
	netPrefix byte

	// TxID is the hash of the transaction fields.
	TxID []byte

	TotalSent ristretto.Scalar
}

func NewStandard(ver uint8, netPrefix byte, fee int64) (*Standard, error) {
	tx := &Standard{
		TxType:    StandardType,
		Version:   ver,
		index:     0,
		netPrefix: netPrefix,
	}

	tx.TotalSent.SetZero()

	// randomly generated nonce - r
	var r ristretto.Scalar
	r.Rand()
	tx.setTxPubKey(r)

	// Set fee
	err := tx.setTxFee(fee)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (s *Standard) setTxPubKey(r ristretto.Scalar) {
	s.r = r
	s.R.ScalarMultBase(&r)
}
func (s *Standard) setTxFee(fee int64) error {
	if fee < 0 {
		return errors.New("fee cannot be negative")
	}
	s.Fee.SetBigInt(big.NewInt(fee))

	return nil
}

func (s *Standard) AddInput(i *Input) error {
	if len(s.Inputs)+1 > maxInputs {
		return errors.New("maximum amount of inputs reached")
	}
	s.Inputs = append(s.Inputs, i)
	return nil
}

func (s *Standard) AddOutput(pubAddr key.PublicAddress, amount ristretto.Scalar) error {
	if len(s.Outputs)+1 > maxOutputs {
		return errors.New("maximum amount of outputs reached")
	}

	pubKey, err := pubAddr.ToKey(s.netPrefix)
	if err != nil {
		return err
	}

	output := NewOutput(s.r, amount, s.index, *pubKey)

	s.Outputs = append(s.Outputs, output)

	s.index = s.index + 1

	s.TotalSent.Add(&s.TotalSent, &amount)

	return nil
}

func (s *Standard) AddDecoys(numMixins int, f FetchDecoys) error {

	if f == nil {
		return errors.New("fetch decoys function cannot be nil")
	}

	for _, input := range s.Inputs {
		decoys := f(numMixins)
		input.Proof.AddDecoys(decoys)
	}
	return nil
}

func (s *Standard) ProveRangeProof() error {

	lenOutputs := len(s.Outputs)
	if lenOutputs < 1 {
		return nil
	}

	// Collect all amounts from outputs
	amounts := make([]ristretto.Scalar, 0, lenOutputs)
	for i := 0; i < lenOutputs; i++ {
		amounts = append(amounts, s.Outputs[i].amount)
	}

	// Create range proof
	proof, err := rangeproof.Prove(amounts, false)
	if err != nil {
		return err
	}

	// XXX: This is not "!=" because the rangeproof will pad when the amount of values does not equal 2^n
	if len(proof.V) < len(amounts) {
		return errors.New("rangeproof did not create proof for all amounts")
	}
	s.RangeProof = proof

	// Move commitment values to their respective outputs
	// along with their blinding factors
	for i := 0; i < lenOutputs; i++ {
		s.Outputs[i].Commitment = proof.V[i].Value
		s.Outputs[i].mask = proof.V[i].BlindingFactor
	}
	return nil
}

func calculateCommToZero(inputs []*Input, outputs []*Output) {

	// Aggregate mask values in each outputs commitment
	var sumOutputMask ristretto.Scalar
	for _, output := range outputs {
		sumOutputMask.Add(&sumOutputMask, &output.mask)
	}

	// Generate len(input)-1 amount of mask values
	// For the pseudoCommitment
	pseudoMaskValues := generateScalars(len(inputs) - 1)

	// Aggregate all mask values
	var sumPseudoMaskValues ristretto.Scalar
	for i := 0; i < len(pseudoMaskValues); i++ {
		sumPseudoMaskValues.Add(&sumPseudoMaskValues, &pseudoMaskValues[i])
	}

	// Append a new mask value to the array of values
	// s.t. it is equal to sumOutputBlinders - sumInputBlinders
	var lastMaskValue ristretto.Scalar
	lastMaskValue.Sub(&sumOutputMask, &sumPseudoMaskValues)
	pseudoMaskValues = append(pseudoMaskValues, lastMaskValue)

	// Calculate and set the commitment to zero for each input
	for i := range inputs {
		input := inputs[i]
		var commToZero ristretto.Scalar
		commToZero.Sub(&pseudoMaskValues[i], &input.mask)

		input.Proof.SetCommToZero(commToZero)
	}

	// Compute Pseudo commitment for each input
	for i := range inputs {
		input := inputs[i]
		pseduoMask := pseudoMaskValues[i]

		pseudoCommitment := CommitAmount(input.amount, pseduoMask)
		input.setPseudoComm(pseudoCommitment)
	}
}

func (s *Standard) encryptOutputValues(encryptValues bool) {
	var zero ristretto.Scalar
	zero.SetZero()
	for i := range s.Outputs {
		output := s.Outputs[i]

		if !encryptValues {
			output.EncryptedAmount = output.amount
			output.EncryptedMask = zero
			continue
		}

		encryptedAmount := EncryptAmount(output.amount, s.r, output.Index, output.viewKey)
		output.EncryptedAmount = encryptedAmount

		encryptedMask := EncryptMask(output.mask, s.r, output.Index, output.viewKey)
		output.EncryptedMask = encryptedMask
	}
}

// Prove creates the rangeproof for output values and creates the mlsag balance and ownership proof
// Prove assumes that all inputs, outputs and decoys have been added to the transaction
func (s *Standard) Prove() error {
	return s.prove(s.CalculateHash, true)
}

func (s *Standard) prove(hasher func() ([]byte, error), encryptValues bool) error {
	// Prove rangeproof, creating the commitments for each output
	err := s.ProveRangeProof()
	if err != nil {
		return err
	}

	// Encrypt mask and amount values
	s.encryptOutputValues(encryptValues)

	// Check that each input has the minimum amount of decoys
	for i := range s.Inputs {
		numDecoys := s.Inputs[i].Proof.LenMembers()
		if numDecoys < minDecoys {
			return fmt.Errorf("each input must contain at least %d decoys input %d contains %d", minDecoys, i, numDecoys)
		}
	}

	// Calculate commitment to zero, adding keys to mlsag
	calculateCommToZero(s.Inputs, s.Outputs)

	// Calculate Hash
	txid, err := hasher()
	if err != nil {
		return err
	}
	// Remove this has from the TxID field - the hash will be changed after the proof
	// is done.
	// TODO: CalculateHash should not set anything, as it's name implies only the
	// calculation of the hash. This should be adjusted and the rest of the code
	// refactored accordingly.
	s.TxID = nil

	// Prove Mlsag
	for i := range s.Inputs {

		// Subtract the pseudo commitment from all of the decoy transactions
		input := s.Inputs[i]
		input.Proof.SubCommToZero(input.PseudoCommitment)
		input.Proof.SetMsg(txid)

		err = input.Prove()
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Standard) CalculateHash() ([]byte, error) {
	if len(s.TxID) != 0 {
		return s.TxID, nil
	}

	buf := new(bytes.Buffer)
	if err := marshalStandard(buf, s, false); err != nil {
		return nil, err
	}

	txid, err := hash.Sha3256(buf.Bytes())
	if err != nil {
		return nil, err
	}

	s.TxID = txid
	return txid, nil
}

func (s *Standard) StandardTx() *Standard {
	return s
}

func (s *Standard) Type() TxType {
	return s.TxType
}

func generateScalars(n int) []ristretto.Scalar {

	var scalars []ristretto.Scalar
	for i := 0; i < n; i++ {
		var x ristretto.Scalar
		x.Rand()
		scalars = append(scalars, x)
	}
	return scalars
}

func CommitAmount(amount, mask ristretto.Scalar) ristretto.Point {

	var blindPoint ristretto.Point
	blindPoint.Derive([]byte("blindPoint"))

	var aH, bG, commitment ristretto.Point
	bG.ScalarMultBase(&mask)
	aH.ScalarMult(&blindPoint, &amount)

	commitment.Add(&aH, &bG)

	return commitment
}

// Equals returns true if two standard tx's are the same
func (s *Standard) Equals(t Transaction) bool {
	other, ok := t.(*Standard)
	if !ok {
		return false
	}

	if s.Version != other.Version {
		return false
	}

	if !bytes.Equal(s.R.Bytes(), other.R.Bytes()) {
		return false
	}

	if !s.Inputs.Equals(other.Inputs) {
		return false
	}

	if !s.Outputs.Equals(other.Outputs) {
		return false
	}

	if !bytes.Equal(s.Fee.Bytes(), other.Fee.Bytes()) {
		return false
	}

	// TxID is not compared, as this could be nil (not set)
	// We could check whether it is set and then check, but
	// the txid is not updated, if a modification is made after
	// calculating the hash. What we can do, is state this edge case and analyse our use-cases.

	return s.RangeProof.Equals(other.RangeProof, true)
}
