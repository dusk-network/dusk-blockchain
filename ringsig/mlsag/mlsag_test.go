package mlsag

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

func TestL(t *testing.T) {
	msg := []byte("hello world")

	var privKey ristretto.Scalar
	privKey.Rand()
	mixin := []ristretto.Point{}

	for i := 0; i < 5; i++ {
		var ranPoint ristretto.Point
		ranPoint.Rand()
		mixin = append(mixin, ranPoint)
	}

	rs := Sign(msg, mixin)

	res := Verify(msg, rs)
	assert.Equal(t, true, res)
	t.Fail()
}
