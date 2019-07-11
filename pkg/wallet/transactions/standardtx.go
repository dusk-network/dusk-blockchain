package transactions

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	wiretx "gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/key"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/mlsag"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof"

	"github.com/bwesterb/go-ristretto"
	"golang.org/x/crypto/sha3"
)

const minDecoys = 7
const maxInputs = 2000
const maxOutputs = 16

type FetchDecoys func(numMixins int) []mlsag.PubKeys

const (
	coinbaseType uint8 = 1
	standardType       = 2
	bidType            = 3
	stakeType          = 4
	timelockType       = 5
)

type StandardTx struct {
	r ristretto.Scalar
	R ristretto.Point

	Inputs  []*Input
	Outputs []*Output
	Fee     ristretto.Scalar

	index     uint32
	netPrefix byte

	RangeProof rangeproof.Proof

	TotalSent ristretto.Scalar
}

func NewStandard(netPrefix byte, fee int64) (*StandardTx, error) {

	tx := &StandardTx{}

	tx.TotalSent.SetZero()

	// Index for subaddresses
	tx.index = 0

	// prefix to signify testnet/mainnet
	tx.netPrefix = netPrefix

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

func (s *StandardTx) setTxPubKey(r ristretto.Scalar) {
	s.r = r
	s.R.ScalarMultBase(&r)
}
func (s *StandardTx) setTxFee(fee int64) error {
	if fee < 0 {
		return errors.New("fee cannot be negative")
	}
	s.Fee.SetBigInt(big.NewInt(fee))

	return nil
}

func (s *StandardTx) AddInput(i *Input) error {
	if len(s.Inputs)+1 > maxInputs {
		return errors.New("maximum amount of inputs reached")
	}
	s.Inputs = append(s.Inputs, i)
	return nil
}

func (s *StandardTx) AddOutput(pubAddr key.PublicAddress, amount ristretto.Scalar) error {
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

func (s *StandardTx) AddDecoys(numMixins int, f FetchDecoys) error {

	if f == nil {
		return errors.New("fetch decoys function cannot be nil")
	}

	for _, input := range s.Inputs {
		decoys := f(numMixins)
		input.Proof.AddDecoys(decoys)
	}
	return nil
}

func (s *StandardTx) ProveRangeProof() error {

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
		s.Outputs[i].setCommitment(proof.V[i].Value)
		s.Outputs[i].setMask(proof.V[i].BlindingFactor)
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

func (s *StandardTx) encryptOutputValues(encryptValues bool) {
	var zero ristretto.Scalar
	zero.SetZero()
	for i := range s.Outputs {
		output := s.Outputs[i]

		if !encryptValues {
			output.setEncryptedAmount(output.amount)
			output.setEncryptedMask(zero)
			continue
		}

		encryptedAmount := EncryptAmount(output.amount, s.r, output.Index, output.viewKey)
		output.setEncryptedAmount(encryptedAmount)

		encryptedMask := EncryptMask(output.mask, s.r, output.Index, output.viewKey)
		output.setEncryptedMask(encryptedMask)
	}
}

// Prove creates the rangeproof for output values and creates the mlsag balance and ownership proof
// Prove assumes that all inputs, outputs and decoys have been added to the transaction
func (s *StandardTx) Prove() error {
	return s.prove(s.Hash, true)
}

func (s *StandardTx) prove(hasher func() ([]byte, error), encryptValues bool) error {
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

func (s *StandardTx) encode(w io.Writer, encodeSignature bool) error {
	err := binary.Write(w, binary.BigEndian, s.R.Bytes())
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, s.Fee.Bytes())
	if err != nil {
		return err
	}
	lenIn := uint64(len(s.Inputs))
	err = binary.Write(w, binary.BigEndian, lenIn)
	if err != nil {
		return err
	}
	for _, input := range s.Inputs {
		err = input.encode(w, encodeSignature)
		if err != nil {
			return err
		}
	}
	lenOut := uint64(len(s.Outputs))
	err = binary.Write(w, binary.BigEndian, lenOut)
	if err != nil {
		return err
	}
	for _, output := range s.Outputs {
		err = output.Encode(w)
		if err != nil {
			return err
		}
	}
	return s.RangeProof.Encode(w, false)
}

func (s *StandardTx) Encode(w io.Writer) error {
	return s.encode(w, true)
}

func (s *StandardTx) Hash() ([]byte, error) {
	return hashBytes(s.encode)
}

func hashBytes(encoder func(io.Writer, bool) error) ([]byte, error) {
	buf := &bytes.Buffer{}
	err := encoder(buf, false)
	if err != nil {
		return nil, err
	}
	hash := sha3.Sum256(buf.Bytes())
	return hash[:], nil
}
func (s *StandardTx) Decode(r io.Reader) error {

	var RBytes [32]byte
	err := binary.Read(r, binary.BigEndian, &RBytes)
	if err != nil {
		return err
	}
	s.R.SetBytes(&RBytes)
	var FeeBytes [32]byte
	err = binary.Read(r, binary.BigEndian, &FeeBytes)
	if err != nil {
		return err
	}
	s.Fee.SetBytes(&FeeBytes)

	var lenIn uint64
	err = binary.Read(r, binary.BigEndian, &lenIn)
	if err != nil {
		return err
	}

	for i := uint64(0); i < lenIn; i++ {
		var input *Input
		err = input.Decode(r)
		if err != nil {
			return err
		}
		s.Inputs = append(s.Inputs, input)
	}

	var lenOut uint64
	err = binary.Read(r, binary.BigEndian, &lenOut)
	if err != nil {
		return err
	}

	for i := uint64(0); i < lenOut; i++ {
		var output *Output
		err = output.Decode(r)
		if err != nil {
			return err
		}
		s.Outputs = append(s.Outputs, output)
	}

	return s.RangeProof.Decode(r, true)
}

func (s *StandardTx) WireStandardTx() (*wiretx.Standard, error) {

	fee := s.Fee.BigInt().Uint64()

	wireTx := wiretx.NewStandard(0, fee, s.R.Bytes())

	// Serialise rangeproof
	buf := &bytes.Buffer{}
	err := s.RangeProof.Encode(buf, true)
	if err != nil {
		return nil, err
	}
	wireTx.RangeProof = buf.Bytes()

	// Serialise inputs
	for i := 0; i < len(s.Inputs); i++ {

		input := s.Inputs[i]

		keyImage := input.KeyImage.Bytes()
		pubKey := input.PubKey.P.Bytes()
		pseudoCommitment := input.PseudoCommitment.Bytes()

		buf = &bytes.Buffer{}
		err = input.Signature.Encode(buf, true)
		if err != nil {
			return nil, err
		}
		sig := buf.Bytes()

		wireInput, err := transactions.NewInput(keyImage, pubKey, pseudoCommitment, sig)
		if err != nil {
			return nil, err
		}
		wireTx.AddInput(wireInput)
	}

	// Serialise outputs
	for i := 0; i < len(s.Outputs); i++ {
		output := s.Outputs[i]

		commitment := output.Commitment.Bytes()
		destKey := output.PubKey.P.Bytes()
		encAmount := output.EncryptedAmount.Bytes()
		encMask := output.EncryptedMask.Bytes()

		wireOutput, err := transactions.NewOutput(commitment, destKey)
		if err != nil {
			return nil, err
		}
		wireOutput.EncryptedAmount = encAmount
		wireOutput.EncryptedMask = encMask

		wireTx.AddOutput(wireOutput)
	}
	return wireTx, nil
}

func (s *StandardTx) Standard() (*StandardTx, error) {
	return s, nil
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
