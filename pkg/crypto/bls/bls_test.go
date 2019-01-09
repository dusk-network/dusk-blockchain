package bls

import (
	"crypto/rand"
	"io"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.dusk.network/dusk-core/bn256"
)

func randomMessage() []byte {
	msg := make([]byte, 32)
	rand.Read(msg)
	return msg
}

// TestSignVerify
func TestSignVerify(t *testing.T) {
	msg := randomMessage()
	pub, priv, err := GenKeyPair(rand.Reader)
	require.NoError(t, err)

	sig, err := Sign(priv, msg)
	require.NoError(t, err)
	require.NoError(t, Verify(pub, msg, sig))

	// Testing that changing the message, the signature is no longer valid
	require.NotNil(t, Verify(pub, randomMessage(), sig))

	// Testing that using a random PK, the signature cannot be verified
	pub2, _, err := GenKeyPair(rand.Reader)
	require.NoError(t, err)
	require.NotNil(t, Verify(pub2, msg, sig))
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

	sig1, err := Sign(priv1, msg1)
	require.NoError(t, err)
	require.NoError(t, Verify(pub1, msg1, sig1))

	sig2, err := Sign(priv2, msg2)
	require.NoError(t, err)
	require.NoError(t, Verify(pub2, msg2, sig2))

	sig3 := Aggregate(sig1, sig2)
	pkeys := []*PublicKey{pub1, pub2}
	require.NoError(t, VerifyBatch(pkeys, [][]byte{msg1, msg2}, sig3))
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
	rogueSignature, err := Sign(sk, msg)
	require.NoError(t, err)

	require.NoError(t, verifyBatch([]*bn256.G2{pub.gx, pk.gx}, [][]byte{msg, msg}, rogueSignature.e, true))
}

func TestMarshalPk(t *testing.T) {
	reader := rand.Reader
	pub, _, err := GenKeyPair(reader)
	require.NoError(t, err)

	pkByteRepr := pub.Marshal()

	g2 := newG2()
	g2.Unmarshal(pkByteRepr)

	g2ByteRepr := g2.Marshal()
	require.Equal(t, pkByteRepr, g2ByteRepr)

	pkInt := new(big.Int).SetBytes(pkByteRepr)
	g2Int := new(big.Int).SetBytes(g2ByteRepr)
	require.Equal(t, pkInt, g2Int)
}

func TestApkVerificationSingleKey(t *testing.T) {
	reader := rand.Reader
	msg := []byte("Get Funky Tonight")

	pub1, priv1, err := GenKeyPair(reader)
	require.NoError(t, err)

	apk := NewApk(pub1)

	signature, err := SafeSign(priv1, pub1, msg)
	require.NoError(t, err)
	require.NoError(t, VerifyApk(apk, msg, signature))
}

func TestApkVerification(t *testing.T) {
	reader := rand.Reader
	msg := []byte("Get Funky Tonight")

	pub1, priv1, err := GenKeyPair(reader)
	require.NoError(t, err)

	pub2, priv2, err := GenKeyPair(reader)
	require.NoError(t, err)

	apk := NewApk(pub1)
	apk.Add(pub2)

	signature, err := SafeSign(priv1, pub1, msg)
	require.NoError(t, err)
	sig2, err := SafeSign(priv2, pub2, msg)
	require.NoError(t, err)

	signature.Aggregate(sig2)
	require.NoError(t, VerifyApk(apk, msg, signature))
}

func TestApkBatchVerification(t *testing.T) {
	reader := rand.Reader
	msg := []byte("Get Funky Tonight")

	pub1, priv1, err := GenKeyPair(reader)
	require.NoError(t, err)

	pub2, priv2, err := GenKeyPair(reader)
	require.NoError(t, err)

	apk := NewApk(pub1)
	apk.Add(pub2)

	sigma, err := SafeSign(priv1, pub1, msg)
	require.NoError(t, err)
	sig2_1, err := SafeSign(priv2, pub2, msg)
	require.NoError(t, err)
	sig := sigma.Aggregate(sig2_1)
	require.NoError(t, VerifyApk(apk, msg, sig))

	msg2 := []byte("Gonna get Shwifty tonight")
	pub3, priv3, err := GenKeyPair(reader)
	require.NoError(t, err)
	apk2 := NewApk(pub2)
	apk2.Add(pub3)
	sig2_2, err := SafeSign(priv2, pub2, msg2)
	require.NoError(t, err)
	sig3_2, err := SafeSign(priv3, pub3, msg2)
	sig2 := sig2_2.Aggregate(sig3_2)
	require.NoError(t, VerifyApk(apk2, msg2, sig2))

	sigma.Aggregate(sig2)
	require.NoError(t, VerifyApkBatch(
		[]*Apk{apk, apk2},
		[][]byte{msg, msg2},
		sigma,
	))
}

func TestCompress(t *testing.T) {
	msg := randomMessage()
	pub, priv, err := GenKeyPair(rand.Reader)
	require.NoError(t, err)

	sig, err := Sign(priv, msg)
	require.NoError(t, err)
	require.NoError(t, Verify(pub, msg, sig))

	sigb := sig.Compress()
	sigTest := &Sig{e: newG1()}
	require.NoError(t, sigTest.Decompress(sigb))

	require.Equal(t, sig.e.Marshal(), sigTest.e.Marshal())

	// require.Equal(t, expected, sig.e.Marshal())
	// check, err := ZipVerify([]*PublicKey{pub}, [][]byte{msg}, sig)
	// require.NoError(t, err)
	// require.Truef(t, check, "The compressed form of the signature does not work")
}

func TestCompressNegative(t *testing.T) {
	sigi, _ := new(big.Int).SetString("4182773077326556947337520923199448906869380994374425209427767883513621864194882999289032108531554108092911166882669083617738348444738725911430569647299733", 10)
	sigbx := sigi.Bytes()
	sig := newG1()
	_, err := sig.Unmarshal(sigbx)

	require.NoError(t, err)

	// require.NoError(t, Verify(pub, msg, sig))

	sigb := sig.Compress()
	decompressedG, err := bn256.Decompress(sigb)
	require.NoError(t, err)
	decompressed := decompressedG.Marshal()
	require.Equal(t, sigbx, decompressed)
}
