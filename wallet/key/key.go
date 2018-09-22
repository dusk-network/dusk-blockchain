package key

import (
	"bytes"

	"github.com/agl/ed25519/edwards25519"
	"github.com/toghrulmaharramov/dusk-go/crypto"
	"github.com/toghrulmaharramov/dusk-go/crypto/base58"
	"github.com/toghrulmaharramov/dusk-go/crypto/hash"
	"github.com/toghrulmaharramov/dusk-go/ec/ed25519"
)

// The choice was made to use slices because of their simplicity
// If it is decided that we should switch to pointers to arrays
// for memory layout or another reason, then this can be changed.
// For now, the performance improvements between the two are insignificant
// While using slices are less bug prone.

const (
	KeySize = 32
)

var (
	netPrefix = byte(0xEF)
)

// Key represents the ed25519 32 byte private/public spend/view keys
type Key struct {
	PrivateSpend []byte // 32 byte
	PublicSpend  []byte // 32 byte
	PrivateView  []byte // 32 byte
	PublicView   []byte // 32 byte
}

func New(seed []byte) (k *Key, err error) {

	PrivateSpend := make([]byte, KeySize)
	PublicSpend := make([]byte, KeySize)
	PrivateView := make([]byte, KeySize)
	PublicView := make([]byte, KeySize)

	// PrivateSpendKey Derivation
	PrivateSpend = seed
	if !ed25519.IsLessThanOrder(PrivateSpend) {
		PrivateSpend, err = ed25519.ScalarRed32(seed)
		if err != nil {
			return nil, err
		}
	}
	// PrivateViewKey Derivation
	PrivateView, err = hash.Keccak256(PrivateSpend)
	if err != nil {
		return nil, err
	}

	PrivateView, err = ed25519.ScalarRed32(PrivateView)
	if !ed25519.IsLessThanOrder(PrivateView) {
		PrivateView, err = ed25519.ScalarRed32(PrivateView)
		if err != nil {
			return nil, err
		}
	}

	// PublicSpendKey Derivation
	PublicSpend = PrivateToPublic(PrivateSpend)

	// PublicViewKey Derivation
	PublicView = PrivateToPublic(PrivateView)

	k = &Key{
		PrivateSpend,
		PublicSpend,
		PrivateView,
		PublicView,
	}
	return k, err
}

// Address will return the base58 encoded stealth address
func (k *Key) Address() (string, error) {

	buf := new(bytes.Buffer)

	err := buf.WriteByte(netPrefix)
	if err != nil {
		return "", err
	}

	_, err = buf.Write(k.PublicSpend)
	if err != nil {
		return "", err
	}

	_, err = buf.Write(k.PublicView)
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

// PrivateToPublic will convert a private spend/view key
// into a public spend/view key
func PrivateToPublic(priv []byte) []byte {

	var A edwards25519.ExtendedGroupElement
	var privBytes [32]byte
	var pubBytes [32]byte

	copy(privBytes[:], priv[:])
	edwards25519.GeScalarMultBase(&A, &privBytes)

	A.ToBytes(&pubBytes)

	return pubBytes[:]
}
