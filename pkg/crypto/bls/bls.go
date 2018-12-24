package bls

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"io"
	"math/big"

	"github.com/cloudflare/bn256"
	"gitlab.dusk.network/pkg/errors"
	"golang.org/x/crypto/sha3"
)

var (
	// g1Str is the hexadecimal string representing the base specified for the G1 base point. It is taken from the cloudflare's bn256 implementation.
	g1Str = "00000000000000000000000000000000000000000000000000000000000000018fb501e34aa387f9aa6fecb86184dc21ee5b88d120b5b59e185cac6c5e089665"

	// g1Base is the base point specified for the G1 group. If one wants to use a
	// different point, set this variable before using any public methods / structs of this package.
	g1Base *bn256.G1

	// g1BaseInt is the big.Int representation of G1Base. This is useful during point compression
	g1BaseInt *big.Int

	// g2Str is the hexadecimal string representing the base specified for the G1 base point.
	g2Str = "012ecca446ff6f3d4d03c76e9b5c752f28bc37b364cb05ac4a37eb32e1c32459708f25386f72c9462b81597d65ae2092c4b97792155dcdaad32b8a6dd41792534c2db10ef5233b0fe3962b9ee6a4bbc2b5bde01a54f3513d42df972e128f31bf12274e5747e8cafacc3716cc8699db79b22f0e4ff3c23e898f694420a3be3087a5"

	// g2Base is the base point specified for the G2 group. If one wants to use a
	// different point, set this variable before using any public methods / structs of this package.
	g2Base *bn256.G2

	// g2BaseInt is the big.Int representation of the G2Base
	g2BaseInt *big.Int
)

// TODO: We should probably transform BLS in a struct and delegate initialization to the
func init() {
	g1Base, g1BaseInt = newG1Base(g1Str)
	g2Base, g2BaseInt = newG2Base(g2Str)
}

// newG1Base is the initialization function for the G1Base point for BN256. It takes as input the HEX string representation of a base point
func newG1Base(strRepr string) (*bn256.G1, *big.Int) {
	buff, err := hex.DecodeString(strRepr)
	if err != nil {
		panic(errors.Wrap(err, "bn256: can't decode base point on G1. Fatal error"))
	}
	g1Base = new(bn256.G1)
	g1BaseInt = new(big.Int).SetBytes(buff)

	_, err = g1Base.Unmarshal(buff)
	if err != nil {
		panic(errors.Wrap(err, "bn256: can't decode base point on G1. Fatal error"))
	}

	return g1Base, g1BaseInt
}

// newG2Base is the initialization function for the G2Base point for BN256
func newG2Base(strRepr string) (*bn256.G2, *big.Int) {
	buff, err := hex.DecodeString(strRepr)
	if err != nil {
		panic(errors.Wrap(err, "bn256: can't decode base point on G2. Fatal error"))
	}
	g2Base = new(bn256.G2)
	g2BaseInt = new(big.Int).SetBytes(buff)

	_, err = g2Base.Unmarshal(buff)
	if err != nil {
		panic(errors.Wrap(err, "bn256: can't decode base point on G2. Fatal error"))
	}

	return g2Base, g2BaseInt
}

// CompactSize is 32bit
const CompactSize = 32

// SecretKey has "x" as secret for the BLS signature
type SecretKey struct {
	x *big.Int
}

// PublicKey is calculated as g^x
type PublicKey struct {
	gx *bn256.G2
}

// Sig is the BLS Signature Struct
type Sig struct {
	e *bn256.G1
}

// NewG1 is the constructor for the G1 group as in the BN256 curve
func NewG1() *bn256.G1 {
	return new(bn256.G1)
}

// NewG2 is the constructor for the G2 group as in the BN256 curve
func NewG2() *bn256.G2 {
	return new(bn256.G2)
}

// GenKeyPair generates Public and Private Keys
func GenKeyPair(randReader io.Reader) (*PublicKey, *SecretKey, error) {
	if randReader == nil {
		randReader = rand.Reader
	}
	x, gx, err := bn256.RandomG2(randReader)

	if err != nil {
		return nil, nil, err
	}

	return &PublicKey{gx}, &SecretKey{x}, nil
}

// Sign the messages
func Sign(key *SecretKey, msg []byte) (*Sig, error) {
	hash, err := hashToPoint(msg)
	if err != nil {
		return nil, err
	}
	p := NewG1()
	p = p.ScalarMult(hash, key.x)
	return &Sig{p}, nil
}

// Verify checks the given BLS signature bls on the message m using the
// public key pkey by verifying that the equality e(H(m), X) == e(H(m), x*B2) ==
// e(x*H(m), B2) == e(S, B2) holds where e is the pairing operation and B2 is
// the base point from curve G2.
func Verify(pkey *PublicKey, msg []byte, signature *Sig) error {
	HM, err := hashToPoint(msg)
	if err != nil {
		return err
	}

	left := bn256.Pair(HM, pkey.gx).Marshal()
	right := bn256.Pair(signature.e, g2Base).Marshal()
	// if subtle.ConstantTimeCompare(left, right) == 1 {
	if !bytes.Equal(left, right) {
		return errors.New("bls: Invalid Signature")
	}

	return nil
}

// Aggregate combines signatures on distinct messages.
// TODO: The messages must be distinct, otherwise the scheme is vulnerable to chosen-key attack.
func Aggregate(one, other *Sig) *Sig {
	res := NewG1()
	res.Add(one.e, other.e)
	return &Sig{e: res}
}

// Batch is a utility function to aggregate distinct messages
// (if not distinct the scheme is vulnerable to chosen-key attack)
func Batch(sigs ...*Sig) (*Sig, error) {
	var sum *Sig
	for i, sig := range sigs {
		if i == 0 {
			sum = sig
		} else {
			sum = Aggregate(sum, sig)
		}
	}

	return sum, nil
}

// VerifyBatch verifies a batch of messages signed with aggregated signature
// the rogue-key attack is prevented by making all messages distinct
func VerifyBatch(pkeys []*PublicKey, msgList [][]byte, signature *Sig, allowDistinct bool) error {
	if !allowDistinct && distinct(msgList) {
		return errors.New("bls: Messages are not distinct")
	}

	var left *bn256.GT
	// TODO: I suspect that this could be sped up by doing the addition through a pool of goroutines
	for i := range msgList {
		h0m, err := hashToPoint(msgList[i])
		if err != nil {
			return err
		}

		if i == 0 {
			left = bn256.Pair(h0m, pkeys[i].gx)
		} else {
			left.Add(left, bn256.Pair(h0m, pkeys[i].gx))
		}
	}

	right := bn256.Pair(signature.e, g2Base)

	//NOTE: Not sure why we could not use subtle.ConstantTimeCompare(left.Marshal(), right.Marshal()) == 1
	if !bytes.Equal(left.Marshal(), right.Marshal()) {
		return errors.New("bls: Invalid Signature")
	}

	return nil
}

// distinct makes sure that the msg list is composed of different messages
func distinct(msgList [][]byte) bool {
	m := make(map[[32]byte]bool)
	for _, msg := range msgList {
		h := sha3.Sum256(msg)
		if m[h] {
			return false
		}
		m[h] = true
	}
	return true
}

// Aggregate is a shortcut for Public Key aggregation
func (pk *PublicKey) Aggregate(pp *PublicKey) *PublicKey {
	p3 := NewG2()
	p3.Add(pk.gx, pp.gx)
	return &PublicKey{p3}
}

// MarshalText encodes the string representation of the public key
func (pk *PublicKey) MarshalText() ([]byte, error) {
	return encodeToText(pk.gx.Marshal()), nil
}

// UnmarshalText decode the string/byte representation into the public key
func (pk *PublicKey) UnmarshalText(data []byte) error {
	bs, err := decodeText(data)
	if err != nil {
		return err
	}
	pk.gx = new(bn256.G2)
	_, err = pk.gx.Unmarshal(bs)
	if err != nil {
		return err
	}
	return nil
}

// MarshalBinary is
func (pk *PublicKey) MarshalBinary() ([]byte, error) {
	return pk.gx.Marshal(), nil
}

// UnmarshalBinary is
func (pk *PublicKey) UnmarshalBinary(data []byte) error {
	pk.gx = new(bn256.G2)
	_, err := pk.gx.Unmarshal(data)
	if err != nil {
		return err
	}
	return nil
}

// H0 is the hash function used to digest a message before mapping it to a
// point.
var H0 = sha3.New256

// hashToPoint is non-deterministic. Hashing to the BN curve should be deterministic to allow for repeatability of the hashing
// TODO: implement the Elligator algorithm for deterministic random-looking hashing to BN256 point. See https://eprint.iacr.org/2014/043.pdf
func hashToPoint(msg []byte) (*bn256.G1, error) {
	h0 := H0()
	_, err := h0.Write(msg)
	if err != nil {
		return nil, err
	}

	hashed := h0.Sum(nil)
	k := new(big.Int).SetBytes(hashed)
	return NewG1().ScalarBaseMult(k), nil
}

func encodeToText(data []byte) []byte {
	buf := make([]byte, base64.RawURLEncoding.EncodedLen(len(data)))
	base64.RawURLEncoding.Encode(buf, data)
	return buf
}

func decodeText(data []byte) ([]byte, error) {
	buf := make([]byte, base64.RawURLEncoding.DecodedLen(len(data)))
	n, err := base64.RawURLEncoding.Decode(buf, data)
	return buf[:n], err
}
