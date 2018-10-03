package ringsig

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"strconv"

	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

// We will use improved MLSAG instead for space saving as we only need c_1
// Scrap this way
// Element is a single ring signature
type Element struct {
	c *ristretto.Scalar
	r *ristretto.Scalar
}

// String conforms to the stringer interface
func (e *Element) String() string {
	c := hex.EncodeToString(e.c.Bytes())
	r := hex.EncodeToString(e.r.Bytes())
	return "c is " + c + " r is " + r
}

// RingSignature is the collection of signatures
type RingSignature []Element

// CreateRingSignature will create a ring sig
// prefixHash is the message
// mixin is the other pubkeys minus owner
// privKey is owner signing

// Returns KeyImage, PubKeys, RingSigs
func CreateRingSignature(prefixHash []byte, mixin []ristretto.Point, privKey ristretto.Scalar) (ristretto.Point, []ristretto.Point, RingSignature) {
	// KeyImage Generation

	// P = xG
	var pubKey ristretto.Point
	var hashedPubKey ristretto.Point
	pubKey.ScalarMultBase(&privKey)
	hashedPubKey.Derive(pubKey.Bytes())

	fmt.Println("We have our Public Key here ", hex.EncodeToString(pubKey.Bytes()))
	fmt.Println("We have our Hashed Public Key here ", hex.EncodeToString(hashedPubKey.Bytes()))
	// XXX : Should we use SHA3-256 instead on raw pub key
	// instead of using the internal hasher in derive?

	// KeyImage = xHp(P)
	var keyImage ristretto.Point
	keyImage.ScalarMult(&hashedPubKey, &privKey)

	// random scalar k
	var k ristretto.Scalar
	k.Rand()

	// Get PubKeys -- for now just make the first one the owner
	pubKeys := make([]ristretto.Point, len(mixin)+1)
	fmt.Println("We have " + strconv.Itoa(len(pubKeys)) + " PubKeys available")
	ownerIndex := rand.Intn(len(pubKeys))
	pubKeys[ownerIndex] = pubKey

	ring := make([]Element, len(pubKeys))

	var sum ristretto.Scalar
	toHash := []byte{}
	// toHash = append(toHash, prefixHash...)

	for i := 0; i < len(pubKeys); i++ {

		if i == ownerIndex {

			// kG
			var kG ristretto.Point
			kG.ScalarMultBase(&k)

			// m || kG
			toHash = append(toHash, kG.Bytes()...)

			// Hp(P) * k
			var samePub ristretto.Point
			samePub = hashedPubKey
			samePub.ScalarMult(&samePub, &k)

			// m || kG || kHp(P)
			toHash = append(toHash, samePub.Bytes()...)
		} else {

			if i > ownerIndex {
				pubKeys[i] = mixin[i-1]
			} else {
				pubKeys[i] = mixin[i]
			}

			c := &ristretto.Scalar{}
			c.Rand()
			r := &ristretto.Scalar{}
			r.Rand()
			ring[i] = Element{
				c: c,
				r: r,
			}

			// cP + rG
			var cP ristretto.Point
			cP = pubKeys[i]
			cP.ScalarMult(&cP, ring[i].c)

			var rG ristretto.Point
			rG.ScalarMultBase(ring[i].r)

			var cPrG ristretto.Point

			cPrG.Add(&cP, &rG)

			toHash = append(toHash, cPrG.Bytes()...)

			// rP + cI
			var rP ristretto.Point
			rP = pubKeys[i]
			rP.ScalarMult(&rP, ring[i].r)

			var cI ristretto.Point
			cI = keyImage
			cI.ScalarMult(&cI, ring[i].c)

			var rPcI ristretto.Point
			rPcI.Add(&rP, &cI)

			toHash = append(toHash, rPcI.Bytes()...)

			// sum = sum + c
			fmt.Println(sum)
			sum.Add(&sum, ring[i].c)
		}
	}
	var h ristretto.Scalar
	h.Derive(toHash)
	fmt.Println("Hash is ", h.BigInt())
	ring[ownerIndex] = Element{
		c: &ristretto.Scalar{},
		r: &ristretto.Scalar{},
	}
	ring[ownerIndex].c.Sub(&h, &sum)
	ring[ownerIndex].r.MulSub(ring[0].c, &privKey, &k) // cp- k XXX: Should this not be k -cp ?

	return keyImage, pubKeys, ring
}

func VerifySignature(prefixHash []byte, keyImage ristretto.Point, pubKeys []ristretto.Point, ringsig []Element) bool {

	// we do not check keyImage if key image is valid, by it being a encoded Ristretto Point
	// should be enough

	if len(pubKeys) != len(ringsig) {
		return false
		fmt.Println("Pubkey and ring sig should have same length")
	}

	sum := ristretto.Scalar{}
	toHash := []byte{}
	// toHash = append(toHash, prefixHash...)
	for i := range ringsig {

		sig := ringsig[i]
		pubKey := pubKeys[i]
		// TODO : checks on c and r

		// cP + rG
		var cP ristretto.Point
		cP = pubKey
		cP.ScalarMult(&cP, sig.c)

		var rG ristretto.Point
		rG.ScalarMultBase(sig.r)

		var cPrG ristretto.Point
		cPrG.Add(&cP, &rG)

		toHash = append(toHash, cPrG.Bytes()...)

		// rH(P) + cI -- Did we hash all pubKeys when signing? May fail
		var rHP ristretto.Point
		rHP.Derive(pubKey.Bytes())

		var cI ristretto.Point
		cI.ScalarMult(&keyImage, sig.c)

		var rHcI ristretto.Point
		rHcI.Add(&rHP, &cI)

		toHash = append(toHash, rHcI.Bytes()...)

		sum.Add(&sum, sig.c)

	}

	var h ristretto.Scalar
	h.Derive(toHash)
	fmt.Println("Derived Hash is ", h.BigInt())
	fmt.Println(h.BigInt())
	fmt.Println(sum.BigInt())

	sum.Sub(&sum, &h)

	var zero ristretto.Scalar
	zero.SetZero()

	var result ristretto.Scalar

	return result.Sub(&sum, &zero).IsNonZeroI() == 0

}
