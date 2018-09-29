package key

import (
	"bytes"
	"errors"

	"github.com/toghrulmaharramov/dusk-go/crypto"
	"github.com/toghrulmaharramov/dusk-go/crypto/base58"
	"github.com/toghrulmaharramov/dusk-go/crypto/hash"

	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

const (
	KeySize = 32
)

var (
	netPrefix = byte(0xEF)
)

// Key represents the ed25519 32 byte private/public spend/view keys
type Key struct {
	PrivateSpend *ristretto.Scalar // 32 byte
	PublicSpend  *ristretto.Point  // 32 byte
	PrivateView  *ristretto.Scalar // 32 byte
	PublicView   *ristretto.Point  // 32 byte
}

// New will generate a new Key from the seed
func New(seed []byte) (k *Key, err error) {

	if len(seed) != 32 {
		return nil, errors.New("Seed size should be 32 bytes")
	}

	// PrivateSpendKey Derivation
	PrivateSpend, err := reduceToScalar(seed)
	if err != nil {
		return nil, err
	}

	// PrivateViewKey Derivation
	pv, err := hash.Sha3256(PrivateSpend.Bytes())
	if err != nil {
		return nil, err
	}

	PrivateView, err := reduceToScalar(pv)
	if err != nil {
		return nil, err
	}

	// PublicSpendKey Derivation
	PublicSpend := privateToPublic(PrivateSpend)

	// PublicViewKey Derivation
	PublicView := privateToPublic(PrivateView)

	k = &Key{
		PrivateSpend,
		PublicSpend,
		PrivateView,
		PublicView,
	}
	return k, err
}

// PublicAddress will return the base58 encoded public address
// The stealth addresses are referred to as the one time
// addresses derived when a user wants to send funds
// to another user
func (k *Key) PublicAddress() (string, error) {

	buf := new(bytes.Buffer)

	err := buf.WriteByte(netPrefix)
	if err != nil {
		return "", err
	}

	_, err = buf.Write(k.PublicSpend.Bytes())
	if err != nil {
		return "", err
	}

	_, err = buf.Write(k.PublicView.Bytes())
	if err != nil {
		return "", err
	}

	checksum, err := crypto.Checksum(buf.Bytes())
	if err != nil {
		return "", err
	}
	_, err = buf.Write(checksum)
	if err != nil {
		return "", err
	}

	return base58.Encode(buf.Bytes()), nil
}

// PubAddrToKey will take a public address
// and return a Key object
func PubAddrToKey(pa string) (*Key, error) {
	return nil, nil
}

// StealthAddress Returns P, R, error
func (k *Key) StealthAddress() (ristretto.Point, ristretto.Point, error) {

	// randomly generated r
	var r ristretto.Scalar
	r.Rand()

	var R ristretto.Point

	R.ScalarMultBase(&r)

	// hash of rA (A = ViewKey)
	var f ristretto.Scalar

	// F = fG
	var F ristretto.Point

	// P = F + B (B = spendKey)
	var P ristretto.Point

	var rA ristretto.Point

	rA.ScalarMult(k.PublicView, &r)

	// f = H(rA)
	f.Derive(rA.Bytes())

	// F = fg
	F.ScalarMultBase(&f)

	// P = F + B
	P.Add(&F, k.PublicSpend)

	return P, R, nil
}

func privateToPublic(s *ristretto.Scalar) *ristretto.Point {
	var p ristretto.Point
	return p.ScalarMultBase(s)
}

func reduceToScalar(s []byte) (*ristretto.Scalar, error) {

	var sc ristretto.Scalar

	if len(s) == 32 {
		var buf [32]byte
		copy(buf[:], s)
		sc.SetReduce32(&buf)
		return &sc, nil
	} else if len(s) == 64 {
		var buf [64]byte
		copy(buf[:], s)
		sc.SetReduced(&buf)
		return &sc, nil
	}

	return nil, errors.New("seed must be of length 32 or 64 bytes")
}
