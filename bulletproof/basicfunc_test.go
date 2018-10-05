package bulletproof

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// This is here to test the matrix operations and commitment scheme
// Will add more extensive tests, for now we are focussing on bulletproof impl.
func TestInnerProd(t *testing.T) {

	k1 := Key{}
	k1[0] = 1
	k1[1] = 1
	k2 := Key{}
	k2[0] = 1
	k2[1] = 1

	k1s := []Key{k1}
	k2s := []Key{k2}

	res := innerProduct(k1s, k2s)
	want := Key{1, 2, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	assert.Equal(t, res, want)
}

func TestHadamard(t *testing.T) {
	k1 := Key{}
	k1[0] = 1
	k1[1] = 1
	k2 := Key{}
	k2[0] = 1
	k2[1] = 1

	k1s := []Key{k1}
	k2s := []Key{k2}

	res := hadamard(k1s, k2s)

	want := []Key{Key{1, 2, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}}
	assert.Equal(t, res, want)
}
