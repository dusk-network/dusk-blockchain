package kadcast

import (
	"math/big"
)

// Computes the XOR distance between 2 different
//ids.
func idXor(a [16]byte, b [16]byte) uint8 {
	distance := [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	i := 0

	for i < 16 {
		distance[i] += a[i] ^ b[i]
		i++
	}
	return collapseDistance(distance)
}

// This function gets the XOR distance as a byte-array
// and collapses it to classify the distance on one of the
// 128 buckets.
func collapseDistance(arr [16]byte) uint8 {
	two := big.NewInt(2)
	base := big.NewInt(10)
	pow := big.NewInt(0)
	collDist := new(big.Int)
	collDist.SetBytes(arr[:])
	println(collDist)
	var i uint8 = 127
	for i > 0 {
		pow.Exp(two, big.NewInt(int64(i)), base)
		println(pow)
		switch collDist.Cmp(pow) {
		// When equals, it goes to this length. (Bucket Lenght ID)
		case 0:
			return i
		// When its greater, it goes to the length of this bucket since
		// we know that it's lower than the previous one
		case 1:
			return i
		default:
			if i == 0 {
				return 0
			}
		}
		i--
	}
	return 0
}

// Performs the hash of the wallet Sk
// and uses it as the ID of a Peer.
func computeIDFromKey() [16]byte {
	panic("uninplemented")
}
