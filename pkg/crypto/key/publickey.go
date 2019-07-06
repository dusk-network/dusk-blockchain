package key

import (
	"bytes"

	"encoding/binary"
	"errors"

	ristretto "github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/base58"
)

// PublicKey represents a pair of PublicSpend and PublicView keys
type PublicKey struct {
	PubSpend *PublicSpend
	PubView  *PublicView
}

// PublicAddress is the encoded prefix + publicSpend + PublicView
type PublicAddress string

func (pa PublicAddress) String() string { return string(pa) }

// ToKey will take a public address and return a Key object with the public parameters filled
func (pa PublicAddress) ToKey(netPrefix byte) (*PublicKey, error) {

	// Base58 Decode
	byt, err := base58.Decode(pa.String())
	if err != nil {
		return nil, err
	}

	var np byte
	var checksum [4]byte
	var publicSpendBytes, publicViewBytes [32]byte

	r := bytes.NewReader(byt)

	binary.Read(r, binary.BigEndian, &np)
	binary.Read(r, binary.BigEndian, &publicSpendBytes)
	binary.Read(r, binary.BigEndian, &publicViewBytes)
	binary.Read(r, binary.BigEndian, &checksum)

	// check net prefix
	if np != netPrefix {
		return nil, errors.New("unrecognised network prefix")
	}

	// compare the checksum
	buf := new(bytes.Buffer)
	buf.WriteByte(np)
	buf.Write(publicSpendBytes[:])
	buf.Write(publicViewBytes[:])

	want := binary.BigEndian.Uint32(checksum[:])
	ok := crypto.CompareChecksum(buf.Bytes(), want)
	if !ok {
		return nil, errors.New("invalid Checksum")
	}

	pubView, err := pubViewFromBytes(publicViewBytes)
	if err != nil {
		return nil, err
	}

	pubSpend, err := pubSpendFromBytes(publicSpendBytes)
	if err != nil {
		return nil, err
	}

	return &PublicKey{
		pubSpend,
		pubView,
	}, nil
}

// StealthAddress Returns P, R
// P = H(r* PubView || Index)G + Pubspend = (H(r * PubView || Index) + privSpend)G
func (k *PublicKey) StealthAddress(r ristretto.Scalar, index uint32) *StealthAddress {

	var rA ristretto.Point
	rA.ScalarMult(k.PubView.point(), &r)

	// f = H(rA || Index)
	rAIndex := concatSlice(rA.Bytes(), uint32ToBytes(index))
	var f ristretto.Scalar
	f.Derive(rAIndex)

	// F = fG
	var F ristretto.Point
	F.ScalarMultBase(&f)

	// P = F + B (B = spendKey)
	var P ristretto.Point
	P.Add(&F, k.PubSpend.point())

	return &StealthAddress{P: P}
}

// PublicAddress will return the base58 encoded public address
// The stealth addresses are referred to as the one time
// addresses derived when a user wants to send funds to another user
func (k *PublicKey) PublicAddress(netPrefix byte) (*PublicAddress, error) {

	buf := new(bytes.Buffer)

	err := buf.WriteByte(netPrefix)
	if err != nil {
		return nil, err
	}

	_, err = buf.Write(k.PubSpend.Bytes())
	if err != nil {
		return nil, err
	}

	_, err = buf.Write(k.PubView.Bytes())
	if err != nil {
		return nil, err
	}

	checksum, err := crypto.Checksum(buf.Bytes())
	if err != nil {
		return nil, err
	}

	cs := make([]byte, 4)
	binary.BigEndian.PutUint32(cs, checksum)

	_, err = buf.Write(cs)
	if err != nil {
		return nil, err
	}

	pubAddrStr, err := base58.Encode(buf.Bytes())
	if err != nil {
		return nil, err
	}

	pubAddr := PublicAddress(pubAddrStr)
	return &pubAddr, nil
}

func uint32ToBytes(x uint32) []byte {
	a := make([]byte, 4)
	binary.BigEndian.PutUint32(a, x)
	return a
}

func concatSlice(slices ...[]byte) []byte {
	var res []byte
	for _, s := range slices {
		res = append(res, s...)
	}
	return res
}
