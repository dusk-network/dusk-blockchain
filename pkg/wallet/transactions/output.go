package transactions

import (
	"encoding/binary"
	"io"

	"github.com/dusk-network/dusk-blockchain/pkg/core/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/crypto/key"

	"github.com/bwesterb/go-ristretto"
)

type Output struct {
	// Commitment to the amount and the mask value
	// This will be generated by the rangeproof
	Commitment ristretto.Point
	amount     ristretto.Scalar
	mask       ristretto.Scalar

	// PubKey refers to the destination key of the receiver
	// One-time pubkey of the receiver
	// Each input will contain a one-time pubkey
	// Only the private key assosciated with this
	// public key can unlock the funds available at this utxo
	PubKey key.StealthAddress

	// Index denotes the position that this output is in the
	// transaction. This is different to the Offset which denotes the
	// position that this output is in, from the start from the blockchain
	Index uint32

	viewKey         key.PublicView
	EncryptedAmount ristretto.Scalar
	EncryptedMask   ristretto.Scalar
}

func NewOutput(r, amount ristretto.Scalar, index uint32, pubKey key.PublicKey) *Output {
	output := &Output{
		amount: amount,
	}

	output.setIndex(index)

	stealthAddr := pubKey.StealthAddress(r, index)
	output.setDestKey(stealthAddr)

	output.viewKey = *pubKey.PubView

	return output
}

func OutputFromWire(out transactions.Output) *Output {

	commitment := sliceToPoint(out.Commitment)
	destKey := sliceToPoint(out.DestKey)
	encAmount := sliceToScalar(out.EncryptedAmount)
	encMask := sliceToScalar(out.EncryptedMask)

	walletOut := &Output{
		Commitment:      commitment,
		EncryptedAmount: encAmount,
		EncryptedMask:   encMask,
		PubKey:          key.StealthAddress{P: destKey},
	}
	return walletOut
}

func sliceToPoint(b []byte) ristretto.Point {
	var bBytes [32]byte
	copy(bBytes[:], b)

	var p ristretto.Point
	p.SetBytes(&bBytes)
	return p
}

func sliceToScalar(b []byte) ristretto.Scalar {
	var bBytes [32]byte
	copy(bBytes[:], b)

	var p ristretto.Scalar
	p.SetBytes(&bBytes)
	return p
}

func (o *Output) Encode(w io.Writer) error {
	err := binary.Write(w, binary.BigEndian, o.Commitment.Bytes())
	if err != nil {
		return err
	}
	// XXX: This can be inferred from the tx layout
	err = binary.Write(w, binary.BigEndian, o.Index)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, o.PubKey.P.Bytes())
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, o.EncryptedAmount.Bytes())
	if err != nil {
		return err
	}
	return binary.Write(w, binary.BigEndian, o.EncryptedMask.Bytes())
}
func (o *Output) Decode(r io.Reader) error {
	var CommBytes [32]byte
	err := binary.Read(r, binary.BigEndian, &CommBytes)
	if err != nil {
		return err
	}
	o.Commitment.SetBytes(&CommBytes)

	err = binary.Read(r, binary.BigEndian, &o.Index)
	if err != nil {
		return err
	}

	var PubKeyBytes [32]byte
	err = binary.Read(r, binary.BigEndian, &PubKeyBytes)
	if err != nil {
		return err
	}
	o.PubKey.P.SetBytes(&PubKeyBytes)

	var EncAmountBytes [32]byte
	err = binary.Read(r, binary.BigEndian, &EncAmountBytes)
	if err != nil {
		return err
	}
	o.EncryptedAmount.SetBytes(&EncAmountBytes)

	var EncMaskBytes [32]byte
	err = binary.Read(r, binary.BigEndian, &EncMaskBytes)
	if err != nil {
		return err
	}
	o.EncryptedMask.SetBytes(&EncMaskBytes)
	return nil
}

func (o *Output) setIndex(index uint32) {
	o.Index = index
}
func (o *Output) setDestKey(stealthAddr *key.StealthAddress) {
	o.PubKey = *stealthAddr
}
func (o *Output) setCommitment(comm ristretto.Point) {
	o.Commitment = comm
}
func (o *Output) setMask(mask ristretto.Scalar) {
	o.mask = mask
}
func (o *Output) setEncryptedAmount(x ristretto.Scalar) {
	o.EncryptedAmount.Set(&x)
}
func (o *Output) setEncryptedMask(x ristretto.Scalar) {
	o.EncryptedMask = x
}

// encAmount = amount + H(H(H(r*PubViewKey || index)))
func EncryptAmount(amount, r ristretto.Scalar, index uint32, pubViewKey key.PublicView) ristretto.Scalar {
	rView := pubViewKey.ScalarMult(r)

	rViewIndex := append(rView.Bytes(), uint32ToBytes(index)...)

	var encryptKey ristretto.Scalar
	encryptKey.Derive(rViewIndex)
	encryptKey.Derive(encryptKey.Bytes())
	encryptKey.Derive(encryptKey.Bytes())

	var encryptedAmount ristretto.Scalar
	encryptedAmount.Add(&amount, &encryptKey)

	return encryptedAmount
}

// decAmount = EncAmount - H(H(H(R*PrivViewKey || index)))
func DecryptAmount(encAmount ristretto.Scalar, R ristretto.Point, index uint32, privViewKey key.PrivateView) ristretto.Scalar {

	var Rview ristretto.Point
	pv := (ristretto.Scalar)(privViewKey)
	Rview.ScalarMult(&R, &pv)

	rViewIndex := append(Rview.Bytes(), uint32ToBytes(index)...)

	var encryptKey ristretto.Scalar
	encryptKey.Derive(rViewIndex)
	encryptKey.Derive(encryptKey.Bytes())
	encryptKey.Derive(encryptKey.Bytes())

	var decryptedAmount ristretto.Scalar
	decryptedAmount.Sub(&encAmount, &encryptKey)

	return decryptedAmount
}

// encMask = mask + H(H(r*PubViewKey || index))
func EncryptMask(mask, r ristretto.Scalar, index uint32, pubViewKey key.PublicView) ristretto.Scalar {
	rView := pubViewKey.ScalarMult(r)
	rViewIndex := append(rView.Bytes(), uint32ToBytes(index)...)

	var encryptKey ristretto.Scalar
	encryptKey.Derive(rViewIndex)
	encryptKey.Derive(encryptKey.Bytes())

	var encryptedMask ristretto.Scalar
	encryptedMask.Add(&mask, &encryptKey)

	return encryptedMask
}

// decMask = Encmask - H(H(r*PubViewKey || index))
func DecryptMask(encMask ristretto.Scalar, R ristretto.Point, index uint32, privViewKey key.PrivateView) ristretto.Scalar {
	var Rview ristretto.Point
	pv := (ristretto.Scalar)(privViewKey)
	Rview.ScalarMult(&R, &pv)

	rViewIndex := append(Rview.Bytes(), uint32ToBytes(index)...)

	var encryptKey ristretto.Scalar
	encryptKey.Derive(rViewIndex)
	encryptKey.Derive(encryptKey.Bytes())

	var decryptedMask ristretto.Scalar
	decryptedMask.Sub(&encMask, &encryptKey)

	return decryptedMask
}

func uint32ToBytes(x uint32) []byte {
	a := make([]byte, 4)
	binary.BigEndian.PutUint32(a, x)
	return a
}
