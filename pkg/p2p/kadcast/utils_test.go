package kadcast

import (
	"testing"

	"gotest.tools/assert"
)

func TestXorDistance(t *testing.T) {
	a := [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	b := [16]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}
	c := [16]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	d := [16]byte{255, 255, 255, 255, 255, 255, 255, 255, 0, 0, 0, 0, 0, 0, 0, 0}
	var max uint16 = 128
	var half uint16 = 64
	var sixteen uint16 = 16
	var zero uint16 = 0
	assert.Equal(t, idXor(a, a), zero)
	assert.Equal(t, idXor(a, c), sixteen)
	assert.Equal(t, idXor(a, b), max)
	assert.Equal(t, idXor(a, d), half)
	assert.Equal(t, idXor(c, d), half)
}
