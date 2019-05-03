package bls

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.dusk.network/dusk-core/bn256"
)

func randomMessage() []byte {
	msg := make([]byte, 32)
	_, _ = rand.Read(msg)
	return msg
}

// TestSignVerify
func TestSignVerify(t *testing.T) {
	msg := randomMessage()
	pub, priv, err := GenKeyPair(rand.Reader)
	require.NoError(t, err)

	sig, err := UnsafeSign(priv, msg)
	require.NoError(t, err)
	require.NoError(t, VerifyUnsafe(pub, msg, sig))

	// Testing that changing the message, the signature is no longer valid
	require.NotNil(t, VerifyUnsafe(pub, randomMessage(), sig))

	// Testing that using a random PK, the signature cannot be verified
	pub2, _, err := GenKeyPair(rand.Reader)
	require.NoError(t, err)
	require.NotNil(t, VerifyUnsafe(pub2, msg, sig))
}

// TestCombine checks for the Batched form of the BLS signature
func TestCombine(t *testing.T) {
	reader := rand.Reader
	msg1 := []byte("Get Funky Tonight")
	msg2 := []byte("Gonna Get Funky Tonight")

	pub1, priv1, err := GenKeyPair(reader)
	require.NoError(t, err)

	pub2, priv2, err := GenKeyPair(reader)
	require.NoError(t, err)

	str1, err := pub1.MarshalText()
	require.NoError(t, err)

	str2, err := pub2.MarshalText()
	require.NoError(t, err)

	require.NotEqual(t, str1, str2)

	sig1, err := UnsafeSign(priv1, msg1)
	require.NoError(t, err)
	require.NoError(t, VerifyUnsafe(pub1, msg1, sig1))

	sig2, err := UnsafeSign(priv2, msg2)
	require.NoError(t, err)
	require.NoError(t, VerifyUnsafe(pub2, msg2, sig2))

	sig3 := UnsafeAggregate(sig1, sig2)
	pkeys := []*PublicKey{pub1, pub2}
	require.NoError(t, VerifyUnsafeBatch(pkeys, [][]byte{msg1, msg2}, sig3))
}

func TestHashToPoint(t *testing.T) {
	msg := []byte("test data")
	g1, err := h0(msg)
	assert.Equal(t, nil, err)
	assert.NotEqual(t, nil, g1)
}

func randomInt(r io.Reader) *big.Int {
	for {
		k, _ := rand.Int(r, bn256.Order)
		if k.Sign() > 0 {
			return k
		}
	}
}
func TestRogueKey(t *testing.T) {
	reader := rand.Reader
	pub, _, err := GenKeyPair(reader)
	require.NoError(t, err)
	// α is the pseudo-secret key of the attacker
	alpha := randomInt(reader)
	// g₂ᵅ
	g2Alpha := newG2().ScalarBaseMult(alpha)

	// pk⁻¹
	rogueGx := newG2()
	rogueGx.Neg(pub.gx)

	pRogue := newG2()
	pRogue.Add(g2Alpha, rogueGx)

	sk, pk := &SecretKey{alpha}, &PublicKey{pRogue}

	msg := []byte("test data")
	rogueSignature, err := UnsafeSign(sk, msg)
	require.NoError(t, err)

	require.NoError(t, verifyBatch([]*bn256.G2{pub.gx, pk.gx}, [][]byte{msg, msg}, rogueSignature.e, true))
}

func TestMarshalPk(t *testing.T) {
	reader := rand.Reader
	pub, _, err := GenKeyPair(reader)
	require.NoError(t, err)

	pkByteRepr := pub.Marshal()

	g2 := newG2()
	_, _ = g2.Unmarshal(pkByteRepr)

	g2ByteRepr := g2.Marshal()
	require.Equal(t, pkByteRepr, g2ByteRepr)

	pkInt := new(big.Int).SetBytes(pkByteRepr)
	g2Int := new(big.Int).SetBytes(g2ByteRepr)
	require.Equal(t, pkInt, g2Int)

	pk, err := UnmarshalPk(pkByteRepr)
	require.NoError(t, err)
	require.Equal(t, pub, pk)
}

func TestApkVerificationSingleKey(t *testing.T) {
	reader := rand.Reader
	msg := []byte("Get Funky Tonight")

	pub1, priv1, err := GenKeyPair(reader)
	require.NoError(t, err)

	apk := NewApk(pub1)

	signature, err := Sign(priv1, pub1, msg)
	require.NoError(t, err)
	require.NoError(t, Verify(apk, msg, signature))

	// testing unmarshalling
	bApk := apk.Marshal()
	uApk, err := UnmarshalApk(bApk)
	assert.NoError(t, err)
	require.NoError(t, Verify(uApk, msg, signature))
}

func TestApkVerification(t *testing.T) {
	reader := rand.Reader
	msg := []byte("Get Funky Tonight")

	pub1, priv1, err := GenKeyPair(reader)
	require.NoError(t, err)

	pub2, priv2, err := GenKeyPair(reader)
	require.NoError(t, err)

	apk := NewApk(pub1)
	assert.NoError(t, apk.Aggregate(pub2))

	signature, err := Sign(priv1, pub1, msg)
	require.NoError(t, err)
	sig2, err := Sign(priv2, pub2, msg)
	require.NoError(t, err)

	signature.Aggregate(sig2)
	require.NoError(t, Verify(apk, msg, signature))
}

func TestApkBatchVerification(t *testing.T) {
	reader := rand.Reader
	msg := []byte("Get Funky Tonight")

	pub1, priv1, err := GenKeyPair(reader)
	require.NoError(t, err)

	pub2, priv2, err := GenKeyPair(reader)
	require.NoError(t, err)

	apk := NewApk(pub1)
	assert.NoError(t, apk.Aggregate(pub2))

	sigma, err := Sign(priv1, pub1, msg)
	require.NoError(t, err)
	sig2_1, err := Sign(priv2, pub2, msg)
	require.NoError(t, err)
	sig := sigma.Aggregate(sig2_1)
	require.NoError(t, Verify(apk, msg, sig))

	msg2 := []byte("Gonna get Shwifty tonight")
	pub3, priv3, err := GenKeyPair(reader)
	require.NoError(t, err)
	apk2 := NewApk(pub2)
	require.NoError(t, apk2.Aggregate(pub3))
	sig2_2, err := Sign(priv2, pub2, msg2)
	require.NoError(t, err)
	sig3_2, err := Sign(priv3, pub3, msg2)
	require.NoError(t, err)
	sig2 := sig2_2.Aggregate(sig3_2)
	require.NoError(t, Verify(apk2, msg2, sig2))

	sigma.Aggregate(sig2)
	require.NoError(t, VerifyBatch(
		[]*Apk{apk, apk2},
		[][]byte{msg, msg2},
		sigma,
	))
}

func TestSafeCompress(t *testing.T) {
	msg := randomMessage()
	pub, priv, err := GenKeyPair(rand.Reader)
	require.NoError(t, err)

	sig, err := Sign(priv, pub, msg)
	require.NoError(t, err)
	require.NoError(t, Verify(NewApk(pub), msg, sig))

	sigb := sig.Compress()
	sigTest := &Signature{e: newG1()}
	require.NoError(t, sigTest.Decompress(sigb))

	require.Equal(t, sig.Marshal(), sigTest.Marshal())
}

func TestUnsafeCompress(t *testing.T) {
	msg := randomMessage()
	pub, priv, err := GenKeyPair(rand.Reader)
	require.NoError(t, err)

	sig, err := UnsafeSign(priv, msg)
	require.NoError(t, err)
	require.NoError(t, VerifyUnsafe(pub, msg, sig))

	sigb := sig.Compress()
	sigTest := &UnsafeSignature{e: newG1()}
	require.NoError(t, sigTest.Decompress(sigb))

	sigM := sig.e.Marshal()
	require.NotEmpty(t, sigM)
	require.Equal(t, sigM, sigTest.e.Marshal())
}

func TestAmbiguousCompress(t *testing.T) {
	msg := randomMessage()
	pub, priv, err := GenKeyPair(rand.Reader)
	require.NoError(t, err)

	sig, err := UnsafeSign(priv, msg)
	require.NoError(t, err)
	require.NoError(t, VerifyUnsafe(pub, msg, sig))

	sigb := sig.Compress()
	require.Equal(t, len(sigb), 33)

	xy1, xy2, err := bn256.DecompressAmbiguous(sigb)
	require.NoError(t, err)

	if xy1 == nil && xy2 == nil {
		fmt.Printf("Original signature: %v\n", new(big.Int).SetBytes(sig.Marshal()).String())
		fmt.Printf("Compressed signature: %v\n", new(big.Int).SetBytes(sigb).String())
		require.Fail(t, "Orcoddue")
	}

	sigM := sig.e.Marshal()
	require.NotEmpty(t, sigM)

	if xy1 != nil {
		sig1 := &UnsafeSignature{xy1}
		if bytes.Equal(sigM, sig1.e.Marshal()) {
			return
		}
	}
	if xy2 != nil {
		sig2 := &UnsafeSignature{xy2}
		if bytes.Equal(sigM, sig2.e.Marshal()) {
			return
		}
	}

	require.Fail(t, "Decompression failed both xy1 and xy2 are nil")
}

func BenchmarkSignature(b *testing.B) {
	msg := randomMessage()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		pk, sk, _ := GenKeyPair(rand.Reader)
		signature, _ := Sign(sk, pk, msg)
		b.StartTimer()
		if err := Verify(NewApk(pk), msg, signature); err != nil {
			panic(err)
		}
	}
}

func BenchmarkUnsafeSignature(b *testing.B) {
	msg := randomMessage()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		pk, sk, _ := GenKeyPair(rand.Reader)
		signature, _ := UnsafeSign(sk, msg)
		b.StartTimer()
		if err := VerifyUnsafe(pk, msg, signature); err != nil {
			panic(err)
		}
	}
}

func BenchmarkVerifyCompressedSignature(b *testing.B) {
	msg := randomMessage()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		pk, sk, _ := GenKeyPair(rand.Reader)
		signature, _ := Sign(sk, pk, msg)
		sigb := signature.Compress()
		apk := NewApk(pk)
		b.StartTimer()

		signature = &Signature{}
		_ = signature.Decompress(sigb)
		_ = Verify(apk, msg, signature)
	}
}

func BenchmarkCompressedUnsafeSignature(b *testing.B) {
	msg := randomMessage()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		pk, sk, _ := GenKeyPair(rand.Reader)
		signature, _ := UnsafeSign(sk, msg)
		sigb := signature.Compress()
		b.StartTimer()
		signature = &UnsafeSignature{}
		_ = signature.Decompress(sigb)
		_ = VerifyUnsafe(pk, msg, signature)
	}
}
