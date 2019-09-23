package transactions

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-crypto/mlsag"
	"github.com/dusk-network/dusk-wallet/key"

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
	// PseudoCommitment refers to a commitment which commits to the same amount
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

// MarshalInput marshals an Input object into a bytes.Buffer.
func MarshalInput(w *bytes.Buffer, i *Input, encodeSignature bool) error {
	if err := encoding.Write256(w, i.KeyImage.Bytes()); err != nil {
		return err
	}

	if err := encoding.Write256(w, i.PubKey.P.Bytes()); err != nil {
		return err
	}

	if err := encoding.Write256(w, i.PseudoCommitment.Bytes()); err != nil {
		return err
	}

	if encodeSignature {
		// Signature needs to be encoded and decoded as VarBytes, to ensure backwards-compatibility with
		// the current testnet.
		buf := new(bytes.Buffer)
		if err := i.Signature.Encode(buf, true); err != nil {
			return err
		}

		return encoding.WriteVarBytes(w, buf.Bytes())
	}
	return nil
}

// Decode an Input object from a bytes.Buffer.
func UnmarshalInput(r *bytes.Buffer, i *Input) error {
	keyImageBytes := make([]byte, 32)
	if err := encoding.Read256(r, keyImageBytes); err != nil {
		return err
	}
	i.KeyImage.UnmarshalBinary(keyImageBytes)

	pubKeyBytes := make([]byte, 32)
	if err := encoding.Read256(r, pubKeyBytes); err != nil {
		return err
	}
	i.PubKey.P.UnmarshalBinary(pubKeyBytes)

	pseudoCommBytes := make([]byte, 32)
	if err := encoding.Read256(r, pseudoCommBytes); err != nil {
		return err
	}
	i.PseudoCommitment.UnmarshalBinary(pseudoCommBytes)

	var sigBytes []byte
	if err := encoding.ReadVarBytes(r, &sigBytes); err != nil {
		return err
	}

	return i.Signature.Decode(bytes.NewBuffer(sigBytes), true)
}

// Equals returns true if two inputs are the same
func (i *Input) Equals(in *Input) bool {
	if in == nil || i == nil {
		return false
	}

	if !bytes.Equal(i.KeyImage.Bytes(), in.KeyImage.Bytes()) {
		return false
	}

	if !bytes.Equal(i.PubKey.P.Bytes(), in.PubKey.P.Bytes()) {
		return false
	}

	if !bytes.Equal(i.PseudoCommitment.Bytes(), in.PseudoCommitment.Bytes()) {
		return false
	}

	return i.Signature.Equals(*in.Signature, false)
}
