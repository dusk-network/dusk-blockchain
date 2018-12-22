package main

/*
	We need:

	- constant time operations
	- private key to public key API
	- hash of arbitrary data to point or scalar API
	- relatively fast
*/

import (
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/ristretto"
)

// func main() {
// 	// GenerateKeyPair()

// 	// x := []byte{0x01, 0x02, 0x03}
// 	// HashToPoint(x)

// 	// CPUIntensive()

// 	// CPUIntensiveGo()
// }

func GenerateKeyPair() {

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

}

func HashToPoint(x []byte) {

	// Derive Point
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
}

func CPUIntensive() {
	// slow me down
	start := time.Now()

	for i := 0; i < 5000; i++ {
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

		newPub.DeriveDalek([]byte{0x01, 0x02, 0x03, 0x04, 0x01, 0x02, 0x03, 0x04})

	}
	elapsed := time.Since(start)
	fmt.Println(elapsed)
}

func CPUIntensiveGo() {
	// slow me down
	start := time.Now()

	wg := sync.WaitGroup{}

	for i := 0; i < 5000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
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

			newPub.DeriveDalek([]byte{0x01, 0x02, 0x03, 0x04, 0x01, 0x02, 0x03, 0x04})
		}()

	}
	elapsed := time.Since(start)
	fmt.Println(elapsed)
}
