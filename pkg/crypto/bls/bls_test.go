package bls

import (
	"crypto/rand"
	"io"
	"math/big"
	"testing"

	"github.com/cloudflare/bn256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, VerifyBatch(pkeys, [][]byte{msg1, msg2}, sig3, false))
}

func TestHashToPoint(t *testing.T) {
	msg := []byte("test data")
	g1, err := hashToPoint(msg)
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

	alpha := randomInt(reader)
	// g₂^alpha
	g2Alpha := NewG2().ScalarBaseMult(alpha)

	// pk^-1
	rogueGx := NewG2()
	rogueGx.Neg(pub.gx)

	pRogue := NewG2()
	pRogue.Add(g2Alpha, rogueGx)

	sk, pk := &SecretKey{alpha}, &PublicKey{pRogue}

	msg := []byte("test data")
	rogueSignature, err := Sign(sk, msg)
	require.NoError(t, err)

	require.NoError(t, VerifyBatch([]*PublicKey{pub, pk}, [][]byte{msg, msg}, rogueSignature, true))
}

/*
func TestCompress(t *testing.T) {
	msg := randomMessage()
	pub, priv, err := GenKeyPair(rand.Reader)
	require.NoError(t, err)

	sig, err := Sign(priv, msg)
	require.NoError(t, err)

	require.NoError(t, Verify(pub, msg, sig))

	check, err := ZipVerify([]*PublicKey{pub}, [][]byte{msg}, sig)
	require.NoError(t, err)
	require.Truef(t, check, "The compressed form of the signature does not work")
}
*/
