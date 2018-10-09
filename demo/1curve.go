package main

/*
	We need:

	- constant time operations
	- private key to public key
	- hash of arbitrary data to point or scalar
	- not too slow
*/

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

func Example() {

	// sK
	var s ristretto.Scalar
	s.Rand()

	// generator
	var gen ristretto.Point
	gen.SetBase()

	//pK = sK * G
	var pubKey ristretto.Point
	pubKey.ScalarMult(&gen, &s)

	fmt.Println("Secret Key:", hex.EncodeToString(s.Bytes()))
	fmt.Println("Generator:", hex.EncodeToString(gen.Bytes()))
	fmt.Println("PubKey:", hex.EncodeToString(pubKey.Bytes()))

	// Derive Point
	x := []byte{0x02, 0x03, 0x04}
	var pDer ristretto.Point
	var pDal ristretto.Point
	pDer.Derive(x)
	pDal.DeriveDalek(x)
	fmt.Println("Derived point:", hex.EncodeToString(pDer.Bytes()))
	fmt.Println("Derived point:", hex.EncodeToString(pDal.Bytes()))

	// Derive Scalar
	var sDer ristretto.Scalar
	sDer.Derive(x)
	fmt.Println("Derived Scalar:", hex.EncodeToString(sDer.Bytes()))

	// slow me down
	start := time.Now()
	for i := 0; i < 3000; i++ {
		var a ristretto.Scalar
		a.Rand()
		var aPub ristretto.Point
		aPub.ScalarMultBase(&a)

		var b ristretto.Scalar
		b.Rand()
		var bPub ristretto.Point
		bPub.ScalarMultBase(&a)

		var ab ristretto.Scalar
		ab.Add(&a, &b)
		var abPub ristretto.Point
		abPub.ScalarMultBase(&ab)

		// a + b + ab
		a.Add(&a, &b).Add(&a, &ab)
		// aPub + bPub + abPub
		aPub.Add(&aPub, &bPub).Add(&aPub, &abPub)

		var newPub ristretto.Point
		newPub.ScalarMultBase(&a)

		newPub.DeriveDalek([]byte("Slow me down"))
	}
	elapsed := time.Since(start)
	fmt.Println(elapsed)
}
