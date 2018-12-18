// This package implements the compact BLS Multisignature construction which preappends the public key to the signature
// according to the *plain public-key model*.
// The form implemented uses an array of distinct keys (as in https://crypto.stanford.edu/~dabo/pubs/papers/BLSmultisig.html)
// instead of the aggregated form (as in https://eprint.iacr.org/2018/483.pdf where {pk₁,...,pkₙ} would be appended to each pkᵢ
//according to apk ← ∏ⁿᵢ₌₁ pk^H₁(pkᵢ, {pk₁,...,pkₙ})

package bls

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/cloudflare/bn256"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
	"gitlab.dusk.network/pkg/errors"
)

// Apk is the short aggregated public key struct
type Apk struct {
	gx *bn256.G2
}

// ApkSig is the plain public key model of the BLS signature
type ApkSig struct {
	e *bn256.G1
}

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

	//TODO: maybe a bit inefficient to recrete G2 instances instead of mutating the underlying group
	return NewG2().ScalarMult(pk.gx, t), nil
}

// NewApk creates an Apk either from a public key or scratch
func NewApk(pk *PublicKey) *Apk {
	if pk == nil {
		return nil
	}

	gx, _ := pkt(pk)
	return &Apk{gx}
}

// Add aggregates a Public Key to the Apk struct
// according to the formula pk^H₁(pkᵢ)
func (apk *Apk) Add(pk *PublicKey) error {
	gxt, err := pkt(pk)
	if err != nil {
		return err
	}

	apk.gx.Add(apk.gx, gxt)
	return nil
}

// AggregateApk aggregates the public key according to the following formula:
// apk ← ∏ⁿᵢ₌₁ pk^H₁(pkᵢ)
func AggregateApk(pks []*PublicKey) *Apk {
	var apk *Apk
	for i, pk := range pks {
		if i == 0 {
			apk = NewApk(pk)
		} else {
			apk.Add(pk)
		}
	}

	return apk
}

// apkWrap turns a BLS Signature into its modified construction
func apkSigWrap(pk *PublicKey, signature *Sig) (*ApkSig, error) {
	// creating tᵢ by hashing PKᵢ
	t, err := h1(pk)
	if err != nil {
		return nil, err
	}

	sigma := NewG1()
	fmt.Printf(
		"Sigma pre-mul %d HEX: %v...\n",
		len(signature.e.Marshal()),
		hex.EncodeToString(signature.e.Marshal()[0:10]),
	)

	sigma.ScalarMult(signature.e, t)
	fmt.Printf(
		"Sigma post-mul %d HEX: %v...\n",
		len(sigma.Marshal()),
		hex.EncodeToString(sigma.Marshal()[0:10]),
	)

	return &ApkSig{e: sigma}, nil
}

// ApkSign creates a signature from the private key and the public key pk
func ApkSign(sk *SecretKey, pk *PublicKey, msg []byte) (*ApkSig, error) {
	sig, err := Sign(sk, msg)
	if err != nil {
		return nil, err
	}

	return apkSigWrap(pk, sig)
}

// Add creates an aggregated signature from a normal BLS Signature and related public key
func (apkSig *ApkSig) Add(pk *PublicKey, sig *Sig) error {
	other, err := apkSigWrap(pk, sig)
	if err != nil {
		return err
	}

	apkSig.Aggregate(other)
	return nil
}

// Aggregate two ApkSig
func (apkSig *ApkSig) Aggregate(other *ApkSig) *ApkSig {
	apkSig.e.Add(apkSig.e, other.e)
	return apkSig
}

// VerifyApk is the verification step of an aggregated apk signature
func VerifyApk(apk *Apk, msg []byte, signature *ApkSig) error {
	return verify(apk.gx, msg, signature.e)
}

// VerifyApkBatch is the verification step of a batch of aggregated apk signatures
// TODO: add the possibility to handle non distinct messages (at batch level after aggregating APK)
func VerifyApkBatch(apks []*Apk, msgs [][]byte, asig *ApkSig) error {
	if len(msgs) != len(apks) {
		msg := fmt.Sprintf(
			"BLS Verify APK Batch: the nr of Public Keys (%d) and the nr. of messages (%d) do not match",
			len(apks),
			len(msgs),
		)
		return errors.New(msg)
	}

	pks := make([]*bn256.G2, len(apks))
	for i, pk := range apks {
		pks[i] = pk.gx
	}

	return verifyBatch(pks, msgs, asig.e, false)
}
