package ristretto_test

import (
	"bytes"
	"fmt"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/ristretto"
)

func Example() {
	// Generate an El-Gamal keypair
	var secretKey ristretto.Scalar
	var publicKey ristretto.Point
	var generator ristretto.Point

	secretKey.Rand() // generate a new secret key
	generator.Derive([]byte("some base point"))
	publicKey.ScalarMult(&generator, &secretKey) // compute public key

	// El-Gamal encrypt a random curve point p into a ciphertext-pair (c1,c2)
	var p ristretto.Point
	var r ristretto.Scalar
	var c1 ristretto.Point
	var c2 ristretto.Point
	p.Rand()
	r.Rand()
	c2.ScalarMult(&generator, &r)
	c1.ScalarMult(&publicKey, &r)
	c1.Add(&c1, &p)

	// Decrypt (c1,c2) back to p
	var blinding, p2 ristretto.Point
	blinding.ScalarMult(&c2, &secretKey)
	p2.Sub(&c1, &blinding)

	fmt.Printf("%v", bytes.Equal(p.Bytes(), p2.Bytes()))
	// Output:
	// true
}
