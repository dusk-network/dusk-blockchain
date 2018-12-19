package bls

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

var initialized bool

func init() {
	// we should wait for BLS to be initialized first
	initialized = G2Base != nil
}

func randomMessage() []byte {
	msg := make([]byte, 32)
	rand.Read(msg)
	return msg
}

// TestSignVerify
// NOTE: Sometimes the test does not succeed due to some mishup in the initialization of the rand.Reader. In these cases, the underline bn256 library yields a segfault since it does not check for the error.
// We should fork bn256 and patch it.
func TestSignVerify(t *testing.T) {
	msg := randomMessage()
	pub, priv, err := GenKeyPair(rand.Reader)
	require.NoError(t, err)

	sig, err := Sign(priv, msg)
	require.NoError(t, err)
	require.NoError(t, Verify(pub, msg, sig))
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
