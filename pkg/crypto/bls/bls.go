package bls

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"math/big"

	"github.com/cloudflare/bn256"
	"golang.org/x/crypto/sha3"
)

// G1Str is the hexadecimal string representing the base specified for the G1
// base point. It is taken from the cloudfare's bn256 implementation.
var G1Str = "00000000000000000000000000000000000000000000000000000000000000018fb501e34aa387f9aa6fecb86184dc21ee5b88d120b5b59e185cac6c5e089665"

// G1Base is the base point specified for the G1 group. If one wants to use a
// different point, set this variable before using any public methods / structs
// of this package.
var G1Base *bn256.G1

// G2Str is the hexadecimal string representing the base specified for the G1
// base point.
var G2Str = "012ecca446ff6f3d4d03c76e9b5c752f28bc37b364cb05ac4a37eb32e1c32459708f25386f72c9462b81597d65ae2092c4b97792155dcdaad32b8a6dd41792534c2db10ef5233b0fe3962b9ee6a4bbc2b5bde01a54f3513d42df972e128f31bf12274e5747e8cafacc3716cc8699db79b22f0e4ff3c23e898f694420a3be3087a5"

// G1BaseInt is the big.Int representation of G1Base. This is useful during point compression
var G1BaseInt *big.Int

// G2Base is the base point specified for the G2 group. If one wants to use a
// different point, set this variable before using any public methods / structs
// of this package.
var G2Base *bn256.G2

func init() {
	buff, err := hex.DecodeString(G1Str)
	if err != nil {
		panic("bn256: can't decode base point on G1. Fatal error")
	}
	G1Base = new(bn256.G1)
	G1BaseInt = new(big.Int).SetBytes(buff)

	_, err = G1Base.Unmarshal(buff)
	if err != nil {
		panic("bn256: can't decode base point on G1. Fatal error")
	}

	buff, err = hex.DecodeString(G2Str)
	if err != nil {
		panic("bn256: can't decode base point on G2. Fatal error.")
	}
	G2Base = new(bn256.G2)
	_, err = G2Base.Unmarshal(buff)
	if err != nil {
		panic("bn256: can't decode base point on G2. Fatal error.")
	}
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
	right := bn256.Pair(signature.e, G2Base).Marshal()
	// if subtle.ConstantTimeCompare(left, right) == 1 {
	if !bytes.Equal(left, right) {
		return errors.New("bls: Invalid Signature")
	}

	return nil
}

// Aggregate combines signatures on distinct messages.
// TODO: The messages must
// be distinct, otherwise the scheme is vulnerable to chosen-key attack.
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
func VerifyBatch(pkeys []*PublicKey, msgList [][]byte, signature *Sig) error {
	if !distinct(msgList) {
		return errors.New("bls: Messages are not distinct")
	}

	var left *bn256.GT
	// TODO: I suspect that this could be sped up by doing the addition through a pool of goroutines
	for i := range msgList {
		HM, err := hashToPoint(msgList[i])
		if err != nil {
			return err
		}

		if i == 0 {
			left = bn256.Pair(HM, pkeys[i].gx)
		} else {
			left.Add(left, bn256.Pair(HM, pkeys[i].gx))
		}
	}

	right := bn256.Pair(signature.e, G2Base)

	//TODO: Not sure why we could not use subtle.ConstantTimeCompare(left.Marshal(), right.Marshal()) == 1
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

// Hash is the hash function used to digest a message before mapping it to a
// point.
var Hash = sha3.New256

func hashToPoint(msg []byte) (*bn256.G1, error) {
	h := Hash()
	_, err := h.Write(msg)
	if err != nil {
		return nil, err
	}

	hashed := h.Sum(nil)

	reader := bytes.NewBuffer(hashed)
	_, HM, err := bn256.RandomG1(reader)
	return HM, err
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
