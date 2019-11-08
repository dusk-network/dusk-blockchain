package kadcast

import (
	"testing"

	"gotest.tools/assert"
)

func TestXorDistance(t *testing.T) {
	a := [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	//b := [16]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}
	//c := [16]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	d := [16]byte{255, 255, 255, 255, 255, 255, 255, 255, 0, 0, 0, 0, 0, 0, 0, 0}
	//var max uint8 = 127
	var half uint8 = 63
	//var zero uint8 = 0
	//assert.Equal(t, idXor(a, a), zero)
	//assert.Equal(t, idXor(a, c), max)
	//assert.Equal(t, idXor(a, b), max)
	assert.Equal(t, idXor(a, d), half)
}
