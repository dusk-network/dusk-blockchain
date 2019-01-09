package bls

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"

	"gitlab.dusk.network/dusk-core/bn256"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
	"gitlab.dusk.network/pkg/errors"
	"golang.org/x/crypto/sha3"
)

var (
	// g1Str is the hexadecimal string representing the base specified for the G1 base point. It is taken from the cloudflare's bn256 implementation.
	g1Str = "00000000000000000000000000000000000000000000000000000000000000018fb501e34aa387f9aa6fecb86184dc21ee5b88d120b5b59e185cac6c5e089665"

	// g1Base is the base point specified for the G1 group. If one wants to use a
	// different point, set this variable before using any public methods / structs of this package.
	g1Base *bn256.G1

	// g2Str is the hexadecimal string representing the base specified for the G1 base point.
	g2Str = "012ecca446ff6f3d4d03c76e9b5c752f28bc37b364cb05ac4a37eb32e1c32459708f25386f72c9462b81597d65ae2092c4b97792155dcdaad32b8a6dd41792534c2db10ef5233b0fe3962b9ee6a4bbc2b5bde01a54f3513d42df972e128f31bf12274e5747e8cafacc3716cc8699db79b22f0e4ff3c23e898f694420a3be3087a5"

	// g2Base is the base point specified for the G2 group. If one wants to use a
	// different point, set this variable before using any public methods / structs of this package.
	g2Base *bn256.G2
)

// TODO: We should probably transform BLS in a struct and delegate initialization to the
func init() {
	g1Base = newG1Base(g1Str)
	g2Base = newG2Base(g2Str)
}

// newG1Base is the initialization function for the G1Base point for BN256. It takes as input the HEX string representation of a base point
func newG1Base(strRepr string) *bn256.G1 {
	buff, err := hex.DecodeString(strRepr)
	if err != nil {
		panic(errors.Wrap(err, "bn256: can't decode base point on G1. Fatal error"))
	}
	g1Base = new(bn256.G1)

	_, err = g1Base.Unmarshal(buff)
	if err != nil {
		panic(errors.Wrap(err, "bn256: can't decode base point on G1. Fatal error"))
	}

	return g1Base
}

// newG2Base is the initialization function for the G2Base point for BN256
func newG2Base(strRepr string) *bn256.G2 {
	buff, err := hex.DecodeString(strRepr)
	if err != nil {
		panic(errors.Wrap(err, "bn256: can't decode base point on G2. Fatal error"))
	}
	g2Base = new(bn256.G2)

	_, err = g2Base.Unmarshal(buff)
	if err != nil {
		panic(errors.Wrap(err, "bn256: can't decode base point on G2. Fatal error"))
	}

	return g2Base
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

// newG1 is the constructor for the G1 group as in the BN256 curve
func newG1() *bn256.G1 {
	return new(bn256.G1)
}

// newG2 is the constructor for the G2 group as in the BN256 curve
func newG2() *bn256.G2 {
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
	hash, err := h0(msg)
	if err != nil {
		return nil, err
	}
	p := newG1()
	p.ScalarMult(hash, key.x)
	return &Sig{p}, nil
}

// Verify checks the given BLS signature bls on the message m using the
// public key pkey by verifying that the equality e(H(m), X) == e(H(m), x*B2) ==
// e(x*H(m), B2) == e(S, B2) holds where e is the pairing operation and B2 is the base point from curve G2.
func Verify(pkey *PublicKey, msg []byte, signature *Sig) error {
	return verify(pkey.gx, msg, signature.e)
}

func verify(pk *bn256.G2, msg []byte, sigma *bn256.G1) error {
	h0m, err := h0(msg)
	if err != nil {
		return err
	}

	pairH0mPK := bn256.Pair(h0m, pk).Marshal()
	pairSigG2 := bn256.Pair(sigma, g2Base).Marshal()
	if subtle.ConstantTimeCompare(pairH0mPK, pairSigG2) != 1 {
		msg := fmt.Sprintf(
			"bls apk: Invalid Signature.\nG1Sig pair (length %d): %v...\nApk H0(m) pair (length %d): %v...",
			len(pairSigG2),
			hex.EncodeToString(pairSigG2[0:10]),
			len(pairH0mPK),
			hex.EncodeToString(pairH0mPK[0:10]),
		)
		return errors.New(msg)
	}

	return nil
}

// Aggregate combines signatures on distinct messages.
// TODO: The messages must be distinct, otherwise the scheme is vulnerable to chosen-key attack.
func Aggregate(one, other *Sig) *Sig {
	res := newG1()
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
	g2s := make([]*bn256.G2, len(pkeys))
	for i, pk := range pkeys {
		g2s[i] = pk.gx
	}
	return verifyBatch(g2s, msgList, signature.e, false)
}

func verifyBatch(pkeys []*bn256.G2, msgList [][]byte, sig *bn256.G1, allowDistinct bool) error {
	if !allowDistinct && !distinct(msgList) {
		return errors.New("bls: Messages are not distinct")
	}

	var pairH0mPKs *bn256.GT
	// TODO: I suspect that this could be sped up by doing the addition through a pool of goroutines
	for i := range msgList {
		h0m, err := h0(msgList[i])
		if err != nil {
			return err
		}

		if i == 0 {
			pairH0mPKs = bn256.Pair(h0m, pkeys[i])
		} else {
			pairH0mPKs.Add(pairH0mPKs, bn256.Pair(h0m, pkeys[i]))
		}
	}

	pairSigG2 := bn256.Pair(sig, g2Base)

	if subtle.ConstantTimeCompare(pairSigG2.Marshal(), pairH0mPKs.Marshal()) != 1 {
		return errors.New("bls: Invalid Signature")
	}

	return nil
}

func verifyCompressed(pks []*bn256.G2, msgList [][]byte, compressedSig []byte, allowDistinct bool) error {
	sig, err := bn256.Decompress(compressedSig)
	if err != nil {
		return err
	}
	return verifyBatch(pks, msgList, sig, allowDistinct)
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
	p3 := newG2()
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
	pk.gx = newG2()
	_, err = pk.gx.Unmarshal(bs)
	if err != nil {
		return err
	}
	return nil
}

// Compress the signature to the 32 byte form
func (sigma *Sig) Compress() []byte {
	return sigma.e.Compress()
}

// Decompress reconstructs the 64 byte signature from the compressed form
func (sigma *Sig) Decompress(x []byte) error {
	e, err := bn256.Decompress(x)
	if err != nil {
		return err
	}
	sigma.e = e
	return nil
}

// Marshal a Signature into a byte array
func (sigma *Sig) Marshal() []byte {
	return sigma.e.Marshal()
}

// Unmarshal a byte array into a Signature
func (sigma *Sig) Unmarshal(msg []byte) error {
	e := newG1()
	if _, err := e.Unmarshal(msg); err != nil {
		return err
	}
	sigma.e = e
	return nil
}

// Marshal returns the binary representation of the G2 point being the public key
func (pk *PublicKey) Marshal() []byte {
	return pk.gx.Marshal()
}

// Unmarshal a public key from a byte array
func (pk *PublicKey) Unmarshal(data []byte) error {
	pk.gx = newG2()
	_, err := pk.gx.Unmarshal(data)
	if err != nil {
		return err
	}
	return nil
}

//hashFn is the hash function used to digest a message before mapping it to a point.
var hashFn = sha3.New256

// TODO: implement the Elligator algorithm for deterministic random-looking hashing to BN256 point. See https://eprint.iacr.org/2014/043.pdf
func h0(msg []byte) (*bn256.G1, error) {
	hashed, err := hash.PerformHash(hashFn(), msg)
	if err != nil {
		return nil, err
	}
	k := new(big.Int).SetBytes(hashed)
	return newG1().ScalarBaseMult(k), nil
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
