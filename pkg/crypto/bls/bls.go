// Package bls implements the compact BLS Multisignature construction which preappends the public key to the signature
// according to the *plain public-key model*.
// The form implemented uses an array of distinct keys (as in https://crypto.stanford.edu/~dabo/pubs/papers/BLSmultisig.html)
// instead of the aggregated form (as in https://eprint.iacr.org/2018/483.pdf where {pk₁,...,pkₙ} would be appended to each pkᵢ
//according to apk ← ∏ⁿᵢ₌₁ pk^H₁(pkᵢ, {pk₁,...,pkₙ})
package bls

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"

	"github.com/pkg/errors"
	"gitlab.dusk.network/dusk-core/bn256"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
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

// TODO: We should probably transform BLS in a struct and delegate initialization to the code user. The motivation is that they might want to use a different curve altogether
func init() {
	g1Base = newG1Base(g1Str)
	g2Base = newG2Base(g2Str)
}

// newG1 is the constructor for the G1 group as in the BN256 curve
func newG1() *bn256.G1 {
	return new(bn256.G1)
}

// newG2 is the constructor for the G2 group as in the BN256 curve
func newG2() *bn256.G2 {
	return new(bn256.G2)
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

// SecretKey has "x" as secret for the BLS signature
type SecretKey struct {
	x *big.Int
}

// PublicKey is calculated as g^x
type PublicKey struct {
	gx *bn256.G2
}

// Apk is the short aggregated public key struct
type Apk struct {
	*PublicKey
}

// Signature is the plain public key model of the BLS signature being resilient to rogue key attack
type Signature struct {
	e *bn256.G1
}

// UnsafeSignature is the BLS Signature Struct not resilient to rogue-key attack
type UnsafeSignature struct {
	e *bn256.G1
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

func UnmarshalPk(b []byte) (*PublicKey, error) {
	pk := &PublicKey{nil}
	if err := pk.Unmarshal(b); err != nil {
		return nil, err
	}
	return pk, nil
}

//hashFn is the hash function used to digest a message before mapping it to a point.
var hashFn = sha3.New256

// h0 is the hash-to-curve-point function
// Hₒ : M -> Gₒ
// TODO: implement the Elligator algorithm for deterministic random-looking hashing to BN256 point. See https://eprint.iacr.org/2014/043.pdf
func h0(msg []byte) (*bn256.G1, error) {
	hashed, err := hash.PerformHash(hashFn(), msg)
	if err != nil {
		return nil, err
	}
	k := new(big.Int).SetBytes(hashed)
	return newG1().ScalarBaseMult(k), nil
}

// h1 is the hashing function used in the modified BLS multi-signature construction
// H₁: G₂->R
func h1(pk *PublicKey) (*big.Int, error) {
	// marshalling G2 into a []byte
	pkb := pk.Marshal()
	// hashing into Z
	h, err := hash.PerformHash(hashFn(), pkb)
	if err != nil {
		return nil, err
	}

	return new(big.Int).SetBytes(h), nil
}

func pkt(pk *PublicKey) (*bn256.G2, error) {
	t, err := h1(pk)
	if err != nil {
		return nil, err
	}

	//TODO: maybe a bit inefficient to recreate G2 instances instead of mutating the underlying group
	return newG2().ScalarMult(pk.gx, t), nil
}

// NewApk creates an Apk either from a public key or scratch
func NewApk(pk *PublicKey) *Apk {
	if pk == nil {
		return nil
	}

	gx, _ := pkt(pk)
	return &Apk{
		PublicKey: &PublicKey{gx},
	}
}

func UnmarshalApk(b []byte) (*Apk, error) {
	apk := &Apk{
		PublicKey: &PublicKey{gx: nil},
	}

	if err := apk.Unmarshal(b); err != nil {
		return nil, err
	}
	return apk, nil
}

// AggregateApk aggregates the public key according to the following formula:
// apk ← ∏ⁿᵢ₌₁ pk^H₁(pkᵢ)
func AggregateApk(pks []*PublicKey) (*Apk, error) {
	var apk *Apk
	for i, pk := range pks {
		if i == 0 {
			apk = NewApk(pk)
			continue
		}

		if err := apk.Aggregate(pk); err != nil {
			return nil, err
		}
	}

	return apk, nil
}

// Aggregate a Public Key to the Apk struct
// according to the formula pk^H₁(pkᵢ)
func (apk *Apk) Aggregate(pk *PublicKey) error {
	gxt, err := pkt(pk)
	if err != nil {
		return err
	}

	apk.gx.Add(apk.gx, gxt)
	return nil
}

// AggregateBytes is a convenient method to aggregate the unmarshalled form of PublicKey directly
func (apk *Apk) AggregateBytes(b []byte) error {
	pk := &PublicKey{}
	if err := pk.Unmarshal(b); err != nil {
		return err
	}
	return apk.Aggregate(pk)
}

// Sign creates a signature from the private key and the public key pk
func Sign(sk *SecretKey, pk *PublicKey, msg []byte) (*Signature, error) {
	sig, err := UnsafeSign(sk, msg)
	if err != nil {
		return nil, err
	}

	return apkSigWrap(pk, sig)
}

func UnmarshalSignature(sig []byte) (*Signature, error) {
	sigma := &Signature{}
	if err := sigma.Unmarshal(sig); err != nil {
		return nil, err
	}
	return sigma, nil
}

// Add creates an aggregated signature from a normal BLS Signature and related public key
func (sigma *Signature) Add(pk *PublicKey, sig *UnsafeSignature) error {
	other, err := apkSigWrap(pk, sig)
	if err != nil {
		return err
	}

	sigma.Aggregate(other)
	return nil
}

func (sigma *Signature) AggregateBytes(other []byte) error {
	sig := &Signature{e: nil}
	if err := sig.Unmarshal(other); err != nil {
		return err
	}
	sigma.Aggregate(sig)
	return nil
}

// Aggregate two Signature
func (sigma *Signature) Aggregate(other *Signature) *Signature {
	sigma.e.Add(sigma.e, other.e)
	return sigma
}

// Compress the signature to the 32 byte form
func (sigma *Signature) Compress() []byte {
	return sigma.e.Compress()
}

// Decompress reconstructs the 64 byte signature from the compressed form
func (sigma *Signature) Decompress(x []byte) error {
	e, err := bn256.Decompress(x)
	if err != nil {
		return err
	}
	sigma.e = e
	return nil
}

// Marshal a Signature into a byte array
func (sigma *Signature) Marshal() []byte {
	return sigma.e.Marshal()
}

// Unmarshal a byte array into a Signature
func (sigma *Signature) Unmarshal(msg []byte) error {
	var err error
	var e *bn256.G1
	if len(msg) == 33 {
		e, err = bn256.Decompress(msg)
		if err != nil {
			return err
		}
		sigma.e = e
		return nil
	}

	e = newG1()
	if _, err := e.Unmarshal(msg); err != nil {
		return err
	}
	sigma.e = e
	return nil
}

// apkSigWrap turns a BLS Signature into its modified construction
func apkSigWrap(pk *PublicKey, signature *UnsafeSignature) (*Signature, error) {
	// creating tᵢ by hashing PKᵢ
	t, err := h1(pk)
	if err != nil {
		return nil, err
	}

	sigma := newG1()

	sigma.ScalarMult(signature.e, t)

	return &Signature{e: sigma}, nil
}

// Verify is the verification step of an aggregated apk signature
func Verify(apk *Apk, msg []byte, sigma *Signature) error {
	return verify(apk.gx, msg, sigma.e)
}

// VerifyBatch is the verification step of a batch of aggregated apk signatures
// TODO: consider adding the possibility to handle non distinct messages (at batch level after aggregating APK)
func VerifyBatch(apks []*Apk, msgs [][]byte, sigma *Signature) error {
	if len(msgs) != len(apks) {
		return fmt.Errorf(
			"BLS Verify APK Batch: the nr of Public Keys (%d) and the nr. of messages (%d) do not match",
			len(apks),
			len(msgs),
		)
	}

	pks := make([]*bn256.G2, len(apks))
	for i, pk := range apks {
		pks[i] = pk.gx
	}

	return verifyBatch(pks, msgs, sigma.e, false)
}

// UnsafeSign generates an UnsafeSignature being vulnerable to the rogue-key attack and therefore can only be used if the messages are distinct
func UnsafeSign(key *SecretKey, msg []byte) (*UnsafeSignature, error) {
	hash, err := h0(msg)
	if err != nil {
		return nil, err
	}
	p := newG1()
	p.ScalarMult(hash, key.x)
	return &UnsafeSignature{p}, nil
}

// Compress the signature to the 32 byte form
func (usig *UnsafeSignature) Compress() []byte {
	return usig.e.Compress()
}

// Decompress reconstructs the 64 byte signature from the compressed form
func (usig *UnsafeSignature) Decompress(x []byte) error {
	e, err := bn256.Decompress(x)
	if err != nil {
		return err
	}
	usig.e = e
	return nil
}

// Marshal an UnsafeSignature into a byte array
func (usig *UnsafeSignature) Marshal() []byte {
	return usig.e.Marshal()
}

// Unmarshal a byte array into an UnsafeSignature
func (usig *UnsafeSignature) Unmarshal(msg []byte) error {
	e := newG1()
	if _, err := e.Unmarshal(msg); err != nil {
		return err
	}
	usig.e = e
	return nil
}

// UnsafeAggregate combines signatures on distinct messages.
func UnsafeAggregate(one, other *UnsafeSignature) *UnsafeSignature {
	res := newG1()
	res.Add(one.e, other.e)
	return &UnsafeSignature{e: res}
}

// UnsafeBatch is a utility function to aggregate distinct messages
// (if not distinct the scheme is vulnerable to chosen-key attack)
func UnsafeBatch(sigs ...*UnsafeSignature) (*UnsafeSignature, error) {
	var sum *UnsafeSignature
	for i, sig := range sigs {
		if i == 0 {
			sum = sig
		} else {
			sum = UnsafeAggregate(sum, sig)
		}
	}

	return sum, nil
}

// VerifyUnsafeBatch verifies a batch of messages signed with aggregated signature
// the rogue-key attack is prevented by making all messages distinct
func VerifyUnsafeBatch(pkeys []*PublicKey, msgList [][]byte, signature *UnsafeSignature) error {
	g2s := make([]*bn256.G2, len(pkeys))
	for i, pk := range pkeys {
		g2s[i] = pk.gx
	}
	return verifyBatch(g2s, msgList, signature.e, false)
}

// VerifyUnsafe checks the given BLS signature bls on the message m using the
// public key pkey by verifying that the equality e(H(m), X) == e(H(m), x*B2) ==
// e(x*H(m), B2) == e(S, B2) holds where e is the pairing operation and B2 is the base point from curve G2.
func VerifyUnsafe(pkey *PublicKey, msg []byte, signature *UnsafeSignature) error {
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

func VerifyCompressed(pks []*bn256.G2, msgList [][]byte, compressedSig []byte, allowDistinct bool) error {
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
