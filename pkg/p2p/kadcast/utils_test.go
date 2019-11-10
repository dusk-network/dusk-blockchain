package kadcast

import (
	"fmt"
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

func TestComputeIDFromKey(t *testing.T) {
	zero := [32]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	hashZero := []byte("f9e2eaaa42d9fe9e558a9b8ef1bf366f190aacaa83bad2641ee106e9041096e4")
	fmt.Printf("Web hash: %x", hashZero)
	var key [16]byte
	copy(key[:], hashZero[0:15])
	fmt.Printf("%x\n", key)
	assert.Equal(t, computeIDFromKey(zero), key)

}
