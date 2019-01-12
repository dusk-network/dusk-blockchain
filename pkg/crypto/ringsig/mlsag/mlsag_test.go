package mlsag

import (
	"math/rand"
	"testing"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
)

func TestRingSig(t *testing.T) {

	msg := []byte("hello world")

	var privKey ristretto.Scalar
	privKey.Rand()
	mixin := []ristretto.Point{}

	for i := 0; i < 5; i++ {
		var ranPoint ristretto.Point
		ranPoint.Rand()
		mixin = append(mixin, ranPoint)
	}

	// Test we get the right number of elements back
	rs := Sign(msg, mixin, privKey)
	assert.Equal(t, len(mixin)+1, len(rs.S))
	assert.Equal(t, len(mixin)+1, len(rs.PubKeys))

	// Test we get back the same pubkeys
	found := make(map[int]struct{})

	var signerPubKey ristretto.Point
	signerPubKey.ScalarMultBase(&privKey)

	for i, key := range rs.PubKeys {
		for _, mix := range mixin {
			if key.Equals(&mix) || key.Equals(&signerPubKey) {
				found[i] = struct{}{}
			}
		}
	}
	assert.Equal(t, len(mixin)+1, len(found))

}

func TestVerify(t *testing.T) {

	for i := 0; i < 100; i++ {
		msg := randBytes(i + 1)
		var privKey ristretto.Scalar
		privKey.Rand()
		mixin := []ristretto.Point{}

		for i := 0; i < 5; i++ {
			var ranPoint ristretto.Point
			ranPoint.Rand()
			mixin = append(mixin, ranPoint)
		}

		// Test we get the right number of elements back
		rs := Sign(msg, mixin, privKey)

		// Test Verify
		res := Verify(msg, rs)
		assert.Equal(t, true, res)

		// Wrong message
		res = Verify(randBytes(5), rs)
		assert.Equal(t, false, res)
	}

}

//https://stackoverflow.com/a/31832326/5203311
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return b
}
