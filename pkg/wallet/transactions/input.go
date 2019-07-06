package transactions

import (
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/key"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/mlsag"

	"github.com/bwesterb/go-ristretto"
)

type Input struct {
	amount, mask ristretto.Scalar
	// One-time pubkey of the receiver
	// Each input will contain a one-time pubkey
	// Only the private key assosciated with this
	// public key can unlock the funds available at this utxo
	PubKey  key.StealthAddress
	privKey ristretto.Scalar
	//PseudoCommitment refers to a commitment which commits to the same amount
	// as our `Commitment` value, however the mask value is changed. This is to be used in proving
	// that the sumInputs-SumOutputs = 0
	PseudoCommitment ristretto.Point
	// Proof is zk proof that proves that the signer knows the one-time pubkey assosciated with
	// an input in the ring. The secondary key in the key-vector is used as an intermediate process
	// for the balance proof. This will use a pseudo-commitment s.t. C = Comm(amount, r)
	// However, in order for the proof to work we need to know all inputs and outputs in the tx.
	// Proof will output a signature and a keyimage
	Proof     *mlsag.DualKey
	Signature *mlsag.Signature
	KeyImage  ristretto.Point
}

/// XXX: maybe we should take an dusk-tx-output and the privkey
// We can then derive the amount, mask from the encrypted amounts
// We can derive the mask from encryptedMask, as it is deterministic
// we can also derive the pubkey
func NewInput(amount, mask, privkey ristretto.Scalar) *Input {

	i := &Input{
		amount:  amount,
		mask:    mask,
		privKey: privkey,
		Proof:   mlsag.NewDualKey(),
	}

	// Set the primary key in the mlsag proof; the key needed to unlock this input
	pubkey := i.Proof.SetPrimaryKey(privkey)

	// Save the pubkey assosciated with the primary key
	// XXX: does the input layer use this anymore? or is it just the mlsag layer
	i.PubKey = key.StealthAddress{P: pubkey}

	return i
}

func (i *Input) setPseudoComm(x ristretto.Point) {
	i.PseudoCommitment = x
}

func (i *Input) Prove() error {
	sig, keyImage, err := i.Proof.Prove()
	if err != nil {
		return err
	}
	i.KeyImage = keyImage
	i.Signature = sig
	return nil
}

func (i *Input) encode(w io.Writer, encodeSignature bool) error {
	err := binary.Write(w, binary.BigEndian, i.PubKey.P.Bytes())
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, i.PseudoCommitment.Bytes())
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, i.KeyImage.Bytes())
	if err != nil {
		return err
	}
	if !encodeSignature {
		return nil
	}
	return i.Signature.Encode(w, false)
}

func (i *Input) Encode(w io.Writer) error {
	return i.encode(w, true)
}

func (i *Input) Decode(r io.Reader) error {
	var PubKeyBytes [32]byte
	err := binary.Read(r, binary.BigEndian, &PubKeyBytes)
	if err != nil {
		return err
	}
	i.PubKey.P.SetBytes(&PubKeyBytes)

	var PseudoBytes [32]byte
	err = binary.Read(r, binary.BigEndian, &PseudoBytes)
	if err != nil {
		return err
	}
	i.PseudoCommitment.SetBytes(&PseudoBytes)

	var KeyImageBytes [32]byte
	err = binary.Read(r, binary.BigEndian, &KeyImageBytes)
	if err != nil {
		return err
	}
	i.KeyImage.SetBytes(&KeyImageBytes)

	return i.Signature.Decode(r, false)
}
