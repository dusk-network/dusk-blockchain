package ringsig_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toghrulmaharramov/dusk-go/ringsig"
	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

func TestCreateSig(t *testing.T) {
	var privKey ristretto.Scalar
	privKey.Rand()
	var prefixHash ristretto.Scalar
	prefixHash.Rand()
	mixin := []ristretto.Point{}

	for i := 0; i < 0; i++ {
		var ranPoint ristretto.Point
		ranPoint.Rand()
		mixin = append(mixin, ranPoint)
	}

	keyI, pubKeys, rs := ringsig.CreateRingSignature(prefixHash.Bytes(), mixin, privKey)
	// for _, pair := range rs {
	// 	fmt.Println(pair.String())
	// }

	assert.Equal(t, true, ringsig.VerifySignature(prefixHash.Bytes(), keyI, pubKeys, rs))
}
