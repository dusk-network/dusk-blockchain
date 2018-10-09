package main

import (
	"fmt"
	"math/big"
	"time"

	"github.com/toghrulmaharramov/dusk-go/ringsig/mlsag"
	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

/*
	We need:
		- Anonymise the sender
		- Do not allow him to double spend and input, when in combination with the pedersons
*/

func main() {
	ringsig()
}
func ringsig() {
	Transaction := []byte{0x8b, 0x65, 0x59, 0x70, 0x15, 0x37, 0x99, 0xaf, 0x2a, 0xea, 0xdc, 0x9f, 0xf1, 0xad, 0xd0, 0xea, 0x6c, 0x72, 0x51, 0xd5, 0x41, 0x54, 0xcf, 0xa9, 0x2c, 0x17, 0x3a, 0x0d, 0xd3, 0x9c, 0x1f, 0x94}

	// Generate private key
	var sK ristretto.Scalar
	sK.SetBigInt(big.NewInt(6))

	mixin := []ristretto.Point{}
	n := 100

	for i := 0; i < n; i++ {
		// Add random public keys
		var ranPoint ristretto.Point
		ranPoint.Rand()
		mixin = append(mixin, ranPoint)
	}

	start := time.Now()
	rs := mlsag.Sign(Transaction, mixin, sK)
	elapsed := time.Since(start)
	fmt.Println("Time taken to Sign ", elapsed)

	start = time.Now()
	res := mlsag.Verify(Transaction, rs)
	elapsed = time.Since(start)
	fmt.Println("Time taken to Verify ", elapsed)

	fmt.Println("Key Image is ", rs.I) // What if we have multiple inputs with different signing keys? General MLSAG

	if res {
		fmt.Println("Signature is valid")
		return
	}
	fmt.Println("Signature is invalid")

}
